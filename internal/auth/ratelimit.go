package auth

import (
	"sync"
	"time"
)

// LoginLimiter bloqueia uma chave (IP + email) depois de muitas tentativas
// erradas dentro de uma janela de tempo. Fica em memória: reiniciar a API zera.
type LoginLimiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	failures map[string][]time.Time
	now      func() time.Time
}

func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{max: max, window: window, failures: make(map[string][]time.Time), now: time.Now}
}

// recent devolve só as falhas que ainda estão dentro da janela (chamar com o lock).
func (l *LoginLimiter) recent(key string) []time.Time {
	cutoff := l.now().Add(-l.window)
	kept := l.failures[key][:0]
	for _, t := range l.failures[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.failures, key)
		return nil
	}
	l.failures[key] = kept
	return kept
}

// Allowed diz se a chave ainda pode tentar logar.
func (l *LoginLimiter) Allowed(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(key)) < l.max
}

// Fail registra uma tentativa errada.
func (l *LoginLimiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures[key] = append(l.recent(key), l.now())
}

// Reset limpa a chave depois de um login certo.
func (l *LoginLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}
