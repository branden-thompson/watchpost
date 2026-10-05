package app

import (
	"reflect"
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/agememo"
)

// lazyMemo is an agememo built on its first use, so a pipeline made as a bare
// literal - every test's - has one; now is its clock, the wall clock when nil.
type lazyMemo[K comparable, V any] struct {
	once sync.Once
	now  func() time.Time
	m    *agememo.Memo[K, V]
}

// memo is the memo, built with o's rules on the first call.
func (l *lazyMemo[K, V]) memo(o agememo.Options) *agememo.Memo[K, V] {
	l.once.Do(func() {
		o.Now = l.now
		l.m = agememo.New[K, V](o)
	})
	return l.m
}

// keptCopy is v with every slice clipped to its length and every map copied,
// so what is added to the copy never reaches v: a kept answer handed out to
// be added to. Derived from the type, so a field added later is covered.
func keptCopy[V any](v V) V {
	rv := reflect.ValueOf(&v).Elem()
	if rv.Kind() != reflect.Struct {
		return v
	}
	for i := range rv.NumField() { // the struct's fields (P10-02)
		f := rv.Field(i)
		if !f.CanSet() {
			continue
		}
		switch f.Kind() {
		case reflect.Slice:
			if !f.IsNil() {
				f.Set(f.Slice3(0, f.Len(), f.Len()))
			}
		case reflect.Map:
			if !f.IsNil() {
				m := reflect.MakeMapWithSize(f.Type(), f.Len())
				for it := f.MapRange(); it.Next(); { // the map's entries (P10-02)
					m.SetMapIndex(it.Key(), it.Value())
				}
				f.Set(m)
			}
		}
	}
	return v
}
