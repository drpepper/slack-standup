require('dotenv').config();
const { App } = require('@slack/bolt');
const session = require('./session');
const { standupBlocks, errorText } = require('./blocks');

const socketMode = !!process.env.SLACK_APP_TOKEN;

const app = new App({
  token: process.env.SLACK_BOT_TOKEN,
  ...(socketMode
    ? { socketMode: true, appToken: process.env.SLACK_APP_TOKEN }
    : { signingSecret: process.env.SLACK_SIGNING_SECRET, port: process.env.PORT || 3000 }),
});

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Extract user IDs from slash command text like "@U123 @U456 plain-name". */
function parseUserMentions(text) {
  // Slack encodes mentions as <@UID> or <@UID|displayname>
  const mentions = [...text.matchAll(/<@([A-Z0-9]+)(?:\|[^>]+)?>/g)].map((m) => m[1]);
  return mentions;
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

  // Filter out bots and the bot user itself
  const profiles = await Promise.all(
    members.map((uid) => client.users.info({ user: uid }).catch(() => null))
  );
  return profiles
    .filter((p) => p?.ok && !p.user.is_bot && !p.user.deleted)
    .map((p) => p.user.id);
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
  const raw = (command.text || '').trim();
  const lower = raw.toLowerCase();

  // ── next ──────────────────────────────────────────────────────────────────
  if (lower === 'next') {
    const sess = session.get(channelId);
    if (!sess) {
      await respond({ text: errorText('No standup in progress. Start one with `/standup`.'), response_type: 'ephemeral' });
      return;
    }
    const updated = session.next(channelId);
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
    const uids = parseUserMentions(raw.slice(4));
    if (uids.length === 0) {
      await respond({ text: errorText('Please mention a user, e.g. `/standup add @alice`.'), response_type: 'ephemeral' });
      return;
    }
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
    const uids = parseUserMentions(raw.slice(7));
    if (uids.length === 0) {
      await respond({ text: errorText('Please mention a user, e.g. `/standup remove @alice`.'), response_type: 'ephemeral' });
      return;
    }
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
  // If there's already a session, warn before overwriting
  if (session.get(channelId)) {
    // Overwrite silently — user explicitly called /standup again
    const ts = messageTs.get(channelId);
    if (ts) {
      // Mark old message as ended
      await updateStandupMessage(client, channelId, ts, null, true).catch(() => {});
    }
    messageTs.delete(channelId);
  }

  let userIds = parseUserMentions(raw);

  if (userIds.length === 0) {
    // No users listed — grab active channel members
    await respond({ text: ':hourglass: Fetching active channel members…', response_type: 'ephemeral' });
    const members = await fetchChannelMembers(client, channelId);
    userIds = await filterActiveMembers(client, members);
    if (userIds.length === 0) {
      await respond({ text: errorText('No active members found in this channel.'), response_type: 'ephemeral' });
      return;
    }
  }

  const sess = session.start(channelId, userIds);
  const msg = await postStandupMessage(client, channelId, sess);
  messageTs.set(channelId, msg.ts);
});

// ---------------------------------------------------------------------------
// Button actions
// ---------------------------------------------------------------------------

app.action('standup_next', async ({ ack, body, client }) => {
  await ack();
  const channelId = body.channel.id;
  const ts = body.message.ts;

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

  session.end(channelId);
  messageTs.delete(channelId);
  await updateStandupMessage(client, channelId, ts, null, true);
});

// ---------------------------------------------------------------------------
// Start
// ---------------------------------------------------------------------------

(async () => {
  await app.start();
  if (socketMode) {
    console.log('⚡ Standup bot running (Socket Mode)');
  } else {
    console.log(`⚡ Standup bot running on port ${process.env.PORT || 3000}`);
  }
})();
