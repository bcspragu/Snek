package main

type CollisionDetector struct {
	collisions map[*snek]*CollisionInfo
}

type CollisionInfo struct {
	attacking  FutureCollision
	attackedBy FutureCollision
}

func (c *CollisionInfo) decrement() {
	c.attacking.decrement()
	c.attackedBy.decrement()
}

type FutureCollision map[*snek]int

func (f FutureCollision) decrement() {
	for s := range f {
		f[s]--
	}
}

func (c *CollisionDetector) Add(attacker, victim *snek, turns int) {
	aInfo, ok := c.collisions[attacker]
	if !ok {
		aInfo = &CollisionInfo{
			attacking:  make(FutureCollision),
			attackedBy: make(FutureCollision),
		}
	}
	// If there was no previous entry, fine. If there was one, overwrite it.
	aInfo.attacking[victim] = turns

	vInfo, ok := c.collisions[victim]
	if !ok {
		vInfo = &CollisionInfo{
			attacking:  make(FutureCollision),
			attackedBy: make(FutureCollision),
		}
	}
	// If there was no previous entry, fine. If there was one, overwrite it.
	vInfo.attackedBy[attacker] = turns
}

// If a snek changed directions, none of our future collisions involving that
// snake are valid any more
func (c *CollisionDetector) Invalidate(s *snek) {
	info, ok := c.collisions[s]
	if !ok {
		// We're done here, there's nothing involving this snek
		return
	}

	// We need to remove ourselves from the attackee lists of these sneks
	for snek := range info.attacking {
		delete(c.collisions[snek].attackedBy, s)
	}

	// We need to remove ourselves from the attacker lists of these sneks
	for snek := range info.attackedBy {
		delete(c.collisions[snek].attacking, s)
	}
}

func (c *CollisionDetector) Advance() {
	for _, info := range c.collisions {
		info.decrement()
	}
}

func (c *CollisionDetector) Died() []*snek {
	sneks := []*snek{}
	for s, info := range c.collisions {
		// Look through each snake's list of collisions, see who's attacking it
		for _, t := range info.attackedBy {
			if t == 0 {
				// If someone is attacking and we've run out of turns until the event
				// happens,
				sneks = append(sneks, s)
			}
		}
	}
	return sneks
}
