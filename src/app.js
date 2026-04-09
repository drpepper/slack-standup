require('dotenv').config();
const { App } = require('@slack/bolt');
const session = require('./session');
const { standupBlocks, errorText } = require('./blocks');

const socketMode = !!process.env.SLACK_APP_TOKEN;

// ---------------------------------------------------------------------------
// Logging
// ---------------------------------------------------------------------------

function log(...args) {
  console.log(new Date().toISOString(), '[standup]', ...args);
}

function logError(...args) {
  console.error(new Date().toISOString(), '[standup][ERROR]', ...args);
}

const app = new App({
  token: process.env.SLACK_BOT_TOKEN,
  ...(socketMode
    ? { socketMode: true, appToken: process.env.SLACK_APP_TOKEN }
    : { signingSecret: process.env.SLACK_SIGNING_SECRET, port: process.env.PORT || 3000 }),
});

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Parse participants from slash command text.
 * Supports @mentions (extracted as Slack user IDs) and plain text names.
 * e.g. "/standup @alice j r" → ['U123ABC', 'j', 'r']
 */
function parseParticipants(text) {
  const participants = [];
  // Replace mentions with their UID and collect them
  const withoutMentions = text.replace(/<@([A-Z0-9]+)(?:\|[^>]+)?>/g, (_, uid) => {
    participants.push(uid);
    return '';
  });
  // Remaining whitespace-separated tokens are plain names
  for (const token of withoutMentions.trim().split(/\s+/)) {
    if (token) participants.push(token);
  }
  return participants;
}

/** Extract only Slack user IDs from @mentions (for add/remove subcommands). */
function parseMentionsOnly(text) {
  return [...text.matchAll(/<@([A-Z0-9]+)(?:\|[^>]+)?>/g)].map((m) => m[1]);
}

/**
 * Fetch all human, non-bot members of a channel.
 * Handles pagination automatically.
 */
async function fetchChannelMembers(client, channelId) {
  const members = [];
  let cursor;
  do {
    const res = await client.conversations.members({ channel: channelId, cursor, limit: 200 });
    members.push(...res.members);
    cursor = res.response_metadata?.next_cursor;
  } while (cursor);

  log(`fetchChannelMembers: ${members.length} raw members in ${channelId}`);

  // Filter out bots and the bot user itself
  const profiles = await Promise.all(
    members.map((uid) => client.users.info({ user: uid }).catch(() => null))
  );
  const human = profiles
    .filter((p) => p?.ok && !p.user.is_bot && !p.user.deleted)
    .map((p) => p.user.id);

  log(`fetchChannelMembers: ${human.length} human members after filtering`);
  return human;
}

/**
 * Fetch active (presence=active) members from a list.
 * Falls back to all members if presence checks fail.
 */
async function filterActiveMembers(client, userIds) {
  const results = await Promise.allSettled(
    userIds.map((uid) => client.users.getPresence({ user: uid }))
  );
  const active = userIds.filter((_, i) => {
    const r = results[i];
    return r.status === 'fulfilled' && r.value.presence === 'active';
  });
  log(`filterActiveMembers: ${active.length}/${userIds.length} active`);
  return active.length > 0 ? active : userIds; // fallback: all members
}

/**
 * Post (or update) the standup message in a channel.
 * Returns { ts, channel } for later updates.
 */
async function postStandupMessage(client, channelId, sess, done = false) {
  return client.chat.postMessage({
    channel: channelId,
    blocks: standupBlocks(sess, done),
    text: done ? 'Standup complete!' : `Standup order started`,
  });
}

async function updateStandupMessage(client, channelId, ts, sess, done = false) {
  return client.chat.update({
    channel: channelId,
    ts,
    blocks: standupBlocks(sess, done),
    text: done ? 'Standup complete!' : 'Standup order',
  });
}

// We store the message ts per channel so we can update it in-place.
const messageTs = new Map(); // channelId -> ts

// ---------------------------------------------------------------------------
// /standup command
// ---------------------------------------------------------------------------
// Subcommands:
//   /standup                       → start with active channel members
//   /standup @user1 @user2 ...     → start with specific users
//   /standup next                  → advance to next speaker
//   /standup add @user             → add user to remaining order
//   /standup remove @user          → remove user from remaining order
//   /standup status                → re-display current order
//   /standup end                   → end the standup
// ---------------------------------------------------------------------------

app.command('/standup', async ({ command, ack, respond, client, say }) => {
  await ack();

  const channelId = command.channel_id;
  const userId = command.user_id;
  const raw = (command.text || '').trim();
  const lower = raw.toLowerCase();

  log(`/standup channel=${channelId} user=${userId} text=${JSON.stringify(raw)}`);

  try {

  // ── next ──────────────────────────────────────────────────────────────────
  if (lower === 'next') {
    const sess = session.get(channelId);
    if (!sess) {
      await respond({ text: errorText('No standup in progress. Start one with `/standup`.'), response_type: 'ephemeral' });
      return;
    }
    const updated = session.next(channelId);
    log(`/standup next: channel=${channelId} remaining=${updated?.remaining?.length ?? 0}`);
    const ts = messageTs.get(channelId);
    if (ts) {
      await updateStandupMessage(client, channelId, ts, updated, !updated);
    } else {
      const msg = await postStandupMessage(client, channelId, updated, !updated);
      if (!updated) messageTs.delete(channelId);
      else messageTs.set(channelId, msg.ts);
    }
    if (!updated) messageTs.delete(channelId);
    return;
  }

  // ── end ───────────────────────────────────────────────────────────────────
  if (lower === 'end') {
    log(`/standup end: channel=${channelId}`);
    session.end(channelId);
    const ts = messageTs.get(channelId);
    messageTs.delete(channelId);
    if (ts) {
      await updateStandupMessage(client, channelId, ts, null, true);
    } else {
      await say({ blocks: standupBlocks(null, true), text: 'Standup ended.' });
    }
    return;
  }

  // ── status ────────────────────────────────────────────────────────────────
  if (lower === 'status') {
    const sess = session.get(channelId);
    if (!sess) {
      await respond({ text: errorText('No standup in progress.'), response_type: 'ephemeral' });
      return;
    }
    log(`/standup status: channel=${channelId}`);
    const msg = await postStandupMessage(client, channelId, sess);
    messageTs.set(channelId, msg.ts);
    return;
  }

  // ── add @user ─────────────────────────────────────────────────────────────
  if (lower.startsWith('add ')) {
    const sess = session.get(channelId);
    if (!sess) {
      await respond({ text: errorText('No standup in progress.'), response_type: 'ephemeral' });
      return;
    }
    const uids = parseParticipants(raw.slice(4));
    if (uids.length === 0) {
      await respond({ text: errorText('Please specify a user, e.g. `/standup add @alice` or `/standup add j`.'), response_type: 'ephemeral' });
      return;
    }
    log(`/standup add: channel=${channelId} users=${uids}`);
    let updated = sess;
    for (const uid of uids) updated = session.add(channelId, uid) ?? updated;
    const ts = messageTs.get(channelId);
    if (ts) await updateStandupMessage(client, channelId, ts, updated);
    else {
      const msg = await postStandupMessage(client, channelId, updated);
      messageTs.set(channelId, msg.ts);
    }
    return;
  }

  // ── remove @user ──────────────────────────────────────────────────────────
  if (lower.startsWith('remove ')) {
    const sess = session.get(channelId);
    if (!sess) {
      await respond({ text: errorText('No standup in progress.'), response_type: 'ephemeral' });
      return;
    }
    const uids = parseParticipants(raw.slice(7));
    if (uids.length === 0) {
      await respond({ text: errorText('Please specify a user, e.g. `/standup remove @alice` or `/standup remove j`.'), response_type: 'ephemeral' });
      return;
    }
    log(`/standup remove: channel=${channelId} users=${uids}`);
    let updated = sess;
    let done = false;
    for (const uid of uids) {
      const result = session.remove(channelId, uid);
      if (result === null) { done = true; break; }
      updated = result;
    }
    const ts = messageTs.get(channelId);
    if (ts) await updateStandupMessage(client, channelId, ts, updated, done);
    else {
      const msg = await postStandupMessage(client, channelId, done ? null : updated, done);
      if (!done) messageTs.set(channelId, msg.ts);
    }
    if (done) messageTs.delete(channelId);
    return;
  }

  // ── start ─────────────────────────────────────────────────────────────────
  log(`/standup start: channel=${channelId}`);

  // If there's already a session, overwrite silently
  if (session.get(channelId)) {
    const ts = messageTs.get(channelId);
    if (ts) {
      await updateStandupMessage(client, channelId, ts, null, true).catch(() => {});
    }
    messageTs.delete(channelId);
  }

  let userIds = parseParticipants(raw);

  if (userIds.length === 0) {
    // No users listed — grab active channel members
    await respond({ text: ':hourglass: Fetching active channel members…', response_type: 'ephemeral' });
    let members;
    try {
      members = await fetchChannelMembers(client, channelId);
    } catch (err) {
      if (err.data?.error === 'not_in_channel') {
        await respond({
          text: ":wave: I'm not in this channel. Please invite me with `/invite @<bot-name>`, then run `/standup` again.",
          response_type: 'ephemeral',
        });
        return;
      }
      throw err;
    }
    userIds = await filterActiveMembers(client, members);
    if (userIds.length === 0) {
      await respond({ text: errorText('No active members found in this channel.'), response_type: 'ephemeral' });
      return;
    }
  }

  log(`/standup start: channel=${channelId} users=${userIds}`);
  const sess = session.start(channelId, userIds);
  let msg;
  try {
    msg = await postStandupMessage(client, channelId, sess);
  } catch (err) {
    if (err.data?.error === 'not_in_channel') {
      await respond({
        text: ":wave: I'm not in this channel. Please invite me with `/invite @<bot-name>`, then run `/standup` again.",
        response_type: 'ephemeral',
      });
      session.end(channelId);
      return;
    }
    throw err;
  }
  messageTs.set(channelId, msg.ts);

  } catch (err) {
    logError(`/standup handler failed:`, err);
    await respond({ text: errorText(`Something went wrong: ${err.message}`), response_type: 'ephemeral' }).catch(() => {});
  }
});

// ---------------------------------------------------------------------------
// Button actions
// ---------------------------------------------------------------------------

app.action('standup_next', async ({ ack, body, client }) => {
  await ack();
  const channelId = body.channel.id;
  const ts = body.message.ts;
  const userId = body.user.id;

  log(`action standup_next: channel=${channelId} user=${userId}`);

  const sess = session.get(channelId);
  if (!sess) {
    // Already done — update to done state
    await updateStandupMessage(client, channelId, ts, null, true);
    return;
  }
  const updated = session.next(channelId);
  await updateStandupMessage(client, channelId, ts, updated, !updated);
  if (!updated) messageTs.delete(channelId);
});

app.action('standup_end', async ({ ack, body, client }) => {
  await ack();
  const channelId = body.channel.id;
  const ts = body.message.ts;
  const userId = body.user.id;

  log(`action standup_end: channel=${channelId} user=${userId}`);

  session.end(channelId);
  messageTs.delete(channelId);
  await updateStandupMessage(client, channelId, ts, null, true);
});

app.error(async (error) => {
  logError('Unhandled error:', error);
});

// ---------------------------------------------------------------------------
// Start
// ---------------------------------------------------------------------------

(async () => {
  log('Starting standup bot…');
  log(`Mode: ${socketMode ? 'Socket Mode' : 'HTTP'}`);
  log(`SLACK_BOT_TOKEN set: ${!!process.env.SLACK_BOT_TOKEN}`);
  log(`SLACK_APP_TOKEN set: ${!!process.env.SLACK_APP_TOKEN}`);
  log(`SLACK_SIGNING_SECRET set: ${!!process.env.SLACK_SIGNING_SECRET}`);
  if (!socketMode) log(`Port: ${process.env.PORT || 3000}`);

  await app.start();

  if (socketMode) {
    log('Bot ready (Socket Mode)');
  } else {
    log(`Bot ready on port ${process.env.PORT || 3000}`);
  }
})().catch((err) => {
  logError('Failed to start:', err);
  process.exit(1);
});
