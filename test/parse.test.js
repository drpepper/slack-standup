const { describe, it } = require('node:test');
const assert = require('node:assert/strict');
const { parseParticipants, parseMentionsOnly } = require('../src/parse');

describe('parseParticipants', () => {
  it('returns empty array for empty string', () => {
    assert.deepStrictEqual(parseParticipants(''), []);
  });

  it('parses plain text names', () => {
    assert.deepStrictEqual(parseParticipants('j r'), ['j', 'r']);
  });

  it('parses a single plain name', () => {
    assert.deepStrictEqual(parseParticipants('alice'), ['alice']);
  });

  it('parses Slack @mentions', () => {
    assert.deepStrictEqual(parseParticipants('<@U123ABC>'), ['U123ABC']);
  });

  it('parses @mentions with display names', () => {
    assert.deepStrictEqual(parseParticipants('<@U123ABC|alice>'), ['U123ABC']);
  });

  it('parses mixed mentions and plain names', () => {
    assert.deepStrictEqual(parseParticipants('<@U123ABC> j r'), ['U123ABC', 'j', 'r']);
  });

  it('handles extra whitespace', () => {
    assert.deepStrictEqual(parseParticipants('  j   r  '), ['j', 'r']);
  });

  it('handles multiple mentions', () => {
    assert.deepStrictEqual(parseParticipants('<@U111> <@U222>'), ['U111', 'U222']);
  });
});

describe('parseMentionsOnly', () => {
  it('returns empty array for plain text', () => {
    assert.deepStrictEqual(parseMentionsOnly('j r'), []);
  });

  it('extracts Slack user IDs', () => {
    assert.deepStrictEqual(parseMentionsOnly('<@U123> <@U456>'), ['U123', 'U456']);
  });

  it('handles mentions with display names', () => {
    assert.deepStrictEqual(parseMentionsOnly('<@U123|alice>'), ['U123']);
  });
});
