package cache

import "testing"

func TestSemanticCacheCollision(t *testing.T) {
	q1 := "SELECT * FROM users WHERE id = 1"
	q2 := "SELECT * FROM users WHERE id = 2"

	key1 := SemanticCacheKey(q1)
	key2 := SemanticCacheKey(q2)

	if key1 == key2 {
		t.Errorf("VULNERABLE: different literal values must produce different cache keys: %d", key1)
	}
}

func TestSemanticCacheEquivalence(t *testing.T) {
	q1 := "SELECT * FROM users WHERE id = 1"
	q2 := "select  *  from  users  where  id  =  1"

	key1 := SemanticCacheKey(q1)
	key2 := SemanticCacheKey(q2)

	if key1 != key2 {
		t.Errorf("semantically equivalent queries should produce the same cache key: %d != %d", key1, key2)
	}
}
