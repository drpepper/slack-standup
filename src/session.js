/**
 * In-memory standup session state, keyed by channel ID.
 *
 * Session shape:
 *   { order: string[], current: number, channelId: string }
 *
 * `order` is the full randomized list of Slack user IDs.
 * `current` is the index of the active speaker (0-based).
 */

const sessions = new Map();

function shuffle(arr) {
  const a = [...arr];
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}

function start(channelId, userIds) {
  const session = { channelId, order: shuffle(userIds), current: 0 };
  sessions.set(channelId, session);
  return session;
}

function get(channelId) {
  return sessions.get(channelId) ?? null;
}

/** Advance to the next speaker. Returns the session, or null if standup is over. */
function next(channelId) {
  const session = sessions.get(channelId);
  if (!session) return null;
  session.current++;
  if (session.current >= session.order.length) {
    sessions.delete(channelId);
    return null;
  }
  return session;
}

/** Add a user after the current speaker. No-op if already in remaining order. */
function add(channelId, userId) {
  const session = sessions.get(channelId);
  if (!session) return null;
  const remaining = session.order.slice(session.current);
  if (remaining.includes(userId)) return session; // already there
  // Insert right after current position
  session.order.splice(session.current + 1, 0, userId);
  return session;
}

/** Remove a user from the remaining order (does not affect already-spoken slots). */
function remove(channelId, userId) {
  const session = sessions.get(channelId);
  if (!session) return null;
  // Only remove from current or later — don't rewrite history
  const idx = session.order.indexOf(userId, session.current);
  if (idx === -1) return session;
  session.order.splice(idx, 1);
  if (session.current >= session.order.length) {
    sessions.delete(channelId);
    return null; // everyone removed, standup over
  }
  return session;
}

function end(channelId) {
  sessions.delete(channelId);
}

module.exports = { start, get, next, add, remove, end };
