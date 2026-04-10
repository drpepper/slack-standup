/**
 * Slack Block Kit builders for standup messages.
 */

/**
 * Build the main standup order message blocks.
 *
 * @param {object} session
 * @param {boolean} [done=false] - standup just finished
 */
function standupBlocks(session, done = false) {
  if (done) {
    return [
      {
        type: 'section',
        text: { type: 'mrkdwn', text: ':tada: *Standup complete!* Everyone has spoken.' },
      },
    ];
  }

  const { order, current } = session;
  const fmt = (id) => /^[A-Z][A-Z0-9]+$/.test(id) ? `<@${id}>` : id;
  const lines = order.map((id, i) => {
    if (i < current) return `:white_check_mark: ~${fmt(id)}~`;
    if (i === current) return `:speaking_head_in_silhouette: *${fmt(id)}* ← up now`;
    return `${i + 1 - current}. ${fmt(id)}`;
  });

  const remaining = order.length - current;

  return [
    {
      type: 'section',
      text: {
        type: 'mrkdwn',
        text: `*Standup order* (${remaining} remaining)\n\n${lines.join('\n')}`,
      },
    },
    { type: 'divider' },
    {
      type: 'actions',
      block_id: 'standup_actions',
      elements: [
        {
          type: 'button',
          text: { type: 'plain_text', text: 'Next ▶' },
          style: 'primary',
          action_id: 'standup_next',
        },
        {
          type: 'button',
          text: { type: 'plain_text', text: 'End standup' },
          style: 'danger',
          action_id: 'standup_end',
          confirm: {
            title: { type: 'plain_text', text: 'End standup?' },
            text: { type: 'mrkdwn', text: 'This will cancel the current standup.' },
            confirm: { type: 'plain_text', text: 'Yes, end it' },
            deny: { type: 'plain_text', text: 'Cancel' },
          },
        },
      ],
    },
  ];
}

/** Ephemeral error message text. */
function errorText(msg) {
  return `:x: ${msg}`;
}

module.exports = { standupBlocks, errorText };
