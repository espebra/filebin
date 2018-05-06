package tokens

import (
	"github.com/dustin/go-humanize"
	"math/rand"
	"sync"
	"time"
)

type Token struct {
	ValidTo         time.Time
	ExpiresReadable string
	Id              string
	VerifiedCount   int
}

type Tokens struct {
	sync.RWMutex
	tokens []Token
}

func Init() Tokens {
	t := Tokens{}
	return t
}

func (t *Tokens) Generate() string {
	t.Cleanup()

	var token Token
	token.Id = RandomString(8)
	now := time.Now().UTC()
	token.ValidTo = now.Add(5 * time.Minute)

	t.Lock()
	t.tokens = append([]Token{token}, t.tokens...)
	t.Unlock()
	return token.Id
}

func (t *Tokens) Verify(token string) bool {
	t.Lock()
	found := false
	now := time.Now().UTC()
	for i, data := range t.tokens {
		if data.Id == token {
			if now.Before(data.ValidTo) {
				t.tokens[i].VerifiedCount = t.tokens[i].VerifiedCount + 1
				found = true
			}
		}
	}
	t.Unlock()
	return found
}

func (t *Tokens) Cleanup() {
	if len(t.tokens) > 250 {
		now := time.Now().UTC()
		t.Lock()
		for i, data := range t.tokens {
			if now.After(data.ValidTo) {
				t.tokens[len(t.tokens)-1], t.tokens[i] = t.tokens[i], t.tokens[len(t.tokens)-1]
			}
		}
		t.Unlock()
	}
}

func (t *Tokens) GetAllTokens() []Token {
	t.RLock()
	defer t.RUnlock()

	var r []Token
	for _, data := range t.tokens {
		data.ExpiresReadable = humanize.Time(data.ValidTo)
		r = append(r, data)
	}
	return r
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
