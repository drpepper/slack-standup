const { describe, it, beforeEach } = require('node:test');
const assert = require('node:assert/strict');
const session = require('../src/session');

describe('session', () => {
  const CH = 'C_TEST';

  beforeEach(() => {
    session.end(CH);
  });

  describe('start', () => {
    it('creates a session with shuffled order', () => {
      const sess = session.start(CH, ['a', 'b', 'c']);
      assert.strictEqual(sess.channelId, CH);
      assert.strictEqual(sess.current, 0);
      assert.strictEqual(sess.order.length, 3);
      assert.deepStrictEqual(sess.order.sort(), ['a', 'b', 'c']);
    });

    it('is retrievable via get', () => {
      session.start(CH, ['a']);
      const sess = session.get(CH);
      assert.ok(sess);
      assert.deepStrictEqual(sess.order, ['a']);
    });
  });

  describe('get', () => {
    it('returns null for unknown channel', () => {
      assert.strictEqual(session.get('C_UNKNOWN'), null);
    });
  });

  describe('next', () => {
    it('advances the current index', () => {
      session.start(CH, ['a', 'b', 'c']);
      const sess = session.next(CH);
      assert.ok(sess);
      assert.strictEqual(sess.current, 1);
    });

    it('returns null when standup is over', () => {
      session.start(CH, ['a']);
      const result = session.next(CH);
      assert.strictEqual(result, null);
      assert.strictEqual(session.get(CH), null);
    });

    it('returns null for unknown channel', () => {
      assert.strictEqual(session.next('C_UNKNOWN'), null);
    });
  });

  describe('add', () => {
    it('inserts user after current speaker', () => {
      session.start(CH, ['a', 'b']);
      // Force a known order for testing
      const sess = session.get(CH);
      sess.order = ['a', 'b'];
      session.add(CH, 'c');
      assert.deepStrictEqual(sess.order, ['a', 'c', 'b']);
    });

    it('is a no-op if user already in remaining order', () => {
      const sess = session.start(CH, ['a', 'b']);
      sess.order = ['a', 'b'];
      session.add(CH, 'b');
      assert.deepStrictEqual(sess.order, ['a', 'b']);
    });

    it('returns null for unknown channel', () => {
      assert.strictEqual(session.add('C_UNKNOWN', 'x'), null);
    });
  });

  describe('remove', () => {
    it('removes user from remaining order', () => {
      const sess = session.start(CH, ['a', 'b', 'c']);
      sess.order = ['a', 'b', 'c'];
      session.remove(CH, 'b');
      assert.deepStrictEqual(sess.order, ['a', 'c']);
    });

    it('returns null when removing the last remaining user', () => {
      const sess = session.start(CH, ['a']);
      sess.order = ['a'];
      const result = session.remove(CH, 'a');
      assert.strictEqual(result, null);
      assert.strictEqual(session.get(CH), null);
    });

    it('is a no-op if user not found', () => {
      const sess = session.start(CH, ['a', 'b']);
      sess.order = ['a', 'b'];
      session.remove(CH, 'z');
      assert.deepStrictEqual(sess.order, ['a', 'b']);
    });

    it('does not remove already-spoken users', () => {
      const sess = session.start(CH, ['a', 'b', 'c']);
      sess.order = ['a', 'b', 'c'];
      session.next(CH); // current = 1 (b is speaking)
      session.remove(CH, 'a'); // a already spoke — should be no-op
      assert.deepStrictEqual(sess.order, ['a', 'b', 'c']);
    });
  });

  describe('end', () => {
    it('removes the session', () => {
      session.start(CH, ['a']);
      session.end(CH);
      assert.strictEqual(session.get(CH), null);
    });

    it('is a no-op for unknown channel', () => {
      session.end('C_UNKNOWN'); // should not throw
    });
  });
});
