package topic

import (
	"strings"
	"sync"
)

func New() *Store {
	s := &Store{
		StoreTrie: &StoreTrie{
			Collections: make([]interface{}, 0),
			Trie:        make(map[string]*StoreTrie),
		},
		Cache: make(map[string][]interface{}),
		Mutex: &sync.RWMutex{},
	}
	s.Separator = "/"
	s.WildcardOne = "+"
	s.WildcardSome = "#"
	return s
}

type StoreTrie struct {
	Collections []interface{}
	Trie        map[string]*StoreTrie
}

type Store struct {
	Separator    string
	WildcardSome string
	WildcardOne  string
	StoreTrie    *StoreTrie
	Mutex        *sync.RWMutex
	Cache        map[string][]interface{}
}

func (slf *Store) Match(Topic string) []interface{} {
	slf.Mutex.RLock()
	defer slf.Mutex.RUnlock()

	var v []interface{}
	return slf.match(v, 0, strings.Split(Topic, slf.Separator), slf.StoreTrie)
}

func (slf *Store) Add(Topic string, Value interface{}) {
	slf.Mutex.Lock()
	defer slf.Mutex.Unlock()
	slf.Cache[Topic] = nil
	slf.add(Value, 0, strings.Split(Topic, slf.Separator), slf.StoreTrie)
}

func (slf *Store) add(Value interface{}, length int, words []string, subTrie *StoreTrie) {
	if length == len(words) {
		subTrie.Collections = append(subTrie.Collections, Value)
		return
	}
	word := words[length]

	var st *StoreTrie
	var ok bool
	if st, ok = subTrie.Trie[word]; !ok {
		subTrie.Trie[word] = &StoreTrie{
			Collections: make([]interface{}, 0),
			Trie:        make(map[string]*StoreTrie),
		}
		st = subTrie.Trie[word]
	}
	slf.add(Value, length+1, words, st)
}

func (slf *Store) match(v []interface{}, length int, words []string, subTrie *StoreTrie) []interface{} {
	if st, ok := subTrie.Trie[slf.WildcardSome]; ok {
		for offset := range st.Collections {
			w := st.Collections[offset]
			if w != slf.Separator {
				for j := length; j < len(words); j++ {
					v = slf.match(v, j, words, st)
				}
				break
			}
		}
		v = slf.match(v, len(words), words, st)
	}

	if length == len(words) {
		v = append(v, subTrie.Collections...)
	} else {
		word := words[length]
		if word != slf.WildcardOne && word != slf.WildcardSome {
			if st, ok := subTrie.Trie[word]; ok {
				v = slf.match(v, length+1, words, st)
			}
		}

		if st, ok := subTrie.Trie[slf.WildcardOne]; ok {
			v = slf.match(v, length+1, words, st)
		}
	}

	return v
}
