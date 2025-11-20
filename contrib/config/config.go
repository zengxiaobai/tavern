package config

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/omalloc/tavern/contrib/log"
)

// Observer is config observer.
type Observer[T any] func(string, *T)

// Config is a config interface.
type Config[T any] interface {
	Scan(v *T) error
	Watch(key string, o Observer[T]) error
	Close() error
}

type config[T any] struct {
	opts   *options
	stop   chan struct{}
	signal chan os.Signal

	observers map[string][]Observer[T]
	bc        *T
}

func New[T any](opts ...Option) Config[T] {
	o := &options{}

	for _, opt := range opts {
		opt(o)
	}

	c := &config[T]{
		opts:      o,
		stop:      make(chan struct{}, 1),
		signal:    make(chan os.Signal, 1),
		observers: make(map[string][]Observer[T]),
		bc:        nil,
	}

	go c.tick()

	return c
}

func (c *config[T]) Scan(v *T) error {
	c.bc = v
	for _, source := range c.opts.sources {
		if files, err := source.Load(); err == nil {
			for _, file := range files {
				unmarshal := toUnmarshal(file.Format)
				if file.Value != nil {
					log.Debugf("[config] load file: %#+v format: %s", file.Key, file.Format)
					if err1 := unmarshal(file.Value, v); err1 != nil {
						log.Errorf("[config] unmarshal file: %#+v error: %s", file.Key, err1)
					}
				}
			}
		} else {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("config file not found: %w", err)
			}
			return err
		}
	}
	return nil
}

func (c *config[T]) Watch(key string, o Observer[T]) error {
	if c.observers[key] == nil {
		c.observers[key] = make([]Observer[T], 0, 8)
	}
	c.observers[key] = append(c.observers[key], o)
	return nil
}

func (c *config[T]) Close() error {
	c.stop <- struct{}{}
	close(c.stop)
	close(c.signal)

	return nil
}

func (c *config[T]) tick() {
	signal.Notify(c.signal, syscall.SIGHUP)

	for {
		select {
		case <-c.stop:
			return
		case <-c.signal:
			log.Debug("[config] received SIGHUP")
			if err := c.Scan(c.bc); err != nil {
				continue
			}
			for k, observers := range c.observers {
				log.Debugf("[config] upgrade key: %s", k)
				for _, observer := range observers {
					observer(k, c.bc)
				}
			}
		}
	}
}
