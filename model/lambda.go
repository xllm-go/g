package model

type puter[Key comparable, Value any] struct {
	put func(Key, Value) Record[Key, Value]
}

type lambdaBuilder[Key comparable, Value any] struct {
	rec Record[Key, Value]
}

func (p puter[Key, Value]) Put(k Key, v Value) Record[Key, Value] {
	return p.put(k, v)
}

func (bdr *lambdaBuilder[Key, Value]) E(str string) *puter[Key, Value] {
	return bdr.B(str == "")
}

func (bdr *lambdaBuilder[Key, Value]) N(inter interface{}) *puter[Key, Value] {
	return bdr.B(inter == nil)
}

func (bdr *lambdaBuilder[Key, Value]) B(condition bool) *puter[Key, Value] {
	if !condition {
		return &puter[Key, Value]{
			put: func(k Key, val Value) Record[Key, Value] { return bdr.rec },
		}
	}
	return &puter[Key, Value]{
		put: func(k Key, v Value) Record[Key, Value] { bdr.rec.Put(k, v); return bdr.rec },
	}
}
