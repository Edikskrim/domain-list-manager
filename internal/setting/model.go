package setting

import "time"

type Setting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Repository interface {
	Create(s *Setting) error
	Get(key string) (*Setting, error)
	List() ([]*Setting, error)
	Update(key string, s *Setting) error
	Delete(key string) error
}
