package model

type puter[Key comparable, Value any] struct {
	put func(Key, Value) Record[Key, Value]
}

type LambdaBuilder[Key comparable, Value any] struct {
	rec Record[Key, Value]
}

func (p puter[Key, Value]) Put(k Key, v Value) Record[Key, Value] {
	return p.put(k, v)
}

func (bdr *LambdaBuilder[Key, Value]) E(str string) *puter[Key, Value] {
	if str == "" {
		return &puter[Key, Value]{
			put: func(k Key, val Value) Record[Key, Value] { return bdr.rec },
		}
	}
	return &puter[Key, Value]{
		put: func(k Key, v Value) Record[Key, Value] { bdr.rec.Put(k, v); return bdr.rec },
	}
}

func (bdr *LambdaBuilder[Key, Value]) N(inter interface{}) *puter[Key, Value] {
	if inter == nil {
		return &puter[Key, Value]{
			put: func(k Key, val Value) Record[Key, Value] { return bdr.rec },
		}
	}
	return &puter[Key, Value]{
		put: func(k Key, v Value) Record[Key, Value] { bdr.rec.Put(k, v); return bdr.rec },
	}
}
