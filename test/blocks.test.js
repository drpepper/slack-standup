const { describe, it } = require('node:test');
const assert = require('node:assert/strict');
const { standupBlocks, errorText } = require('../src/blocks');

describe('standupBlocks', () => {
  it('returns done message when done=true', () => {
    const blocks = standupBlocks(null, true);
    assert.strictEqual(blocks.length, 1);
    assert.ok(blocks[0].text.text.includes('Standup complete'));
  });

  it('renders Slack user IDs as mentions', () => {
    const sess = { order: ['U123ABC', 'U456DEF'], current: 0 };
    const blocks = standupBlocks(sess);
    const text = blocks[0].text.text;
    assert.ok(text.includes('<@U123ABC>'), 'should mention first user');
    assert.ok(text.includes('<@U456DEF>'), 'should mention second user');
  });

  it('renders plain text names as-is', () => {
    const sess = { order: ['alice', 'bob'], current: 0 };
    const blocks = standupBlocks(sess);
    const text = blocks[0].text.text;
    assert.ok(text.includes('alice'));
    assert.ok(text.includes('bob'));
    assert.ok(!text.includes('<@alice>'), 'plain names should not be wrapped in mentions');
  });

  it('shows current speaker indicator', () => {
    const sess = { order: ['a', 'b'], current: 0 };
    const blocks = standupBlocks(sess);
    const text = blocks[0].text.text;
    assert.ok(text.includes('*a*'), 'current speaker should be bold');
    assert.ok(text.includes('up now'));
  });

  it('shows checkmark for completed speakers', () => {
    const sess = { order: ['a', 'b', 'c'], current: 1 };
    const blocks = standupBlocks(sess);
    const text = blocks[0].text.text;
    assert.ok(text.includes(':white_check_mark:'));
    assert.ok(text.includes('~a~'), 'completed speaker should be struck through');
  });

  it('shows remaining count', () => {
    const sess = { order: ['a', 'b', 'c'], current: 1 };
    const blocks = standupBlocks(sess);
    const text = blocks[0].text.text;
    assert.ok(text.includes('2 remaining'));
  });

  it('includes action buttons', () => {
    const sess = { order: ['a'], current: 0 };
    const blocks = standupBlocks(sess);
    const actions = blocks.find((b) => b.type === 'actions');
    assert.ok(actions);
    const actionIds = actions.elements.map((e) => e.action_id);
    assert.ok(actionIds.includes('standup_next'));
    assert.ok(actionIds.includes('standup_end'));
  });

  it('handles mixed Slack IDs and plain names', () => {
    const sess = { order: ['U123', 'alice', 'U456'], current: 0 };
    const blocks = standupBlocks(sess);
    const text = blocks[0].text.text;
    assert.ok(text.includes('<@U123>'));
    assert.ok(text.includes('alice'));
    assert.ok(!text.includes('<@alice>'));
    assert.ok(text.includes('<@U456>'));
  });
});

describe('errorText', () => {
  it('prefixes with :x: emoji', () => {
    assert.strictEqual(errorText('oops'), ':x: oops');
  });
});
