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

module.exports = { parseParticipants, parseMentionsOnly };
