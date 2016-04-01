package art

import "errors"

type iteratorLevel struct {
	node     *artNode
	childIdx int
}

type iterator struct {
	version int // tree version

	tree       *tree
	nextNode   *artNode
	depthLevel int
	depth      []*iteratorLevel
}

func (t *tree) ForEach(callback Callback) {
	t.forEach(t.root, callback)
}

func (t *tree) forEach(current *artNode, callback Callback) {
	if current == nil {
		return
	}

	if !callback(current) {
		return
	}

	switch current.kind {
	case NODE_4:
		node := current.node4()
		for i, limit := 0, len(node.children); i < limit; i++ {
			child := node.children[i]
			if child != nil {
				t.forEach(child, callback)
			}
		}

	case NODE_16:
		node := current.node16()
		for i, limit := 0, len(node.children); i < limit; i++ {
			child := node.children[i]
			if child != nil {
				t.forEach(child, callback)
			}
		}

	case NODE_48:
		node := current.node48()
		for i, limit := 0, len(node.keys); i < limit; i++ {
			idx := node.keys[byte(i)]
			if idx <= 0 {
				continue
			}
			child := node.children[idx-1]
			if child != nil {
				t.forEach(child, callback)
			}
		}

	case NODE_256:
		node := current.node256()
		for i, limit := 0, len(node.children); i < limit; i++ {
			child := node.children[i]
			if child != nil {
				t.forEach(child, callback)
			}
		}
	}
}

func (t *tree) ForEachPrefix(key Key, callback Callback) {
	t.forEachPrefix(t.root, key, callback)
}

func (t *tree) forEachPrefix(current *artNode, key Key, callback Callback) {
	if current == nil {
		return
	}

	depth := 0
	for current != nil {
		if current.isLeaf() {
			leaf := current.leaf()

			if leaf.match(key) {
				callback(current)
			}
			return
		}

		if depth == len(key) {
			leaf := current.minimum()
			if leaf.match(key) {
				t.forEach(current, callback)
			}

			return
		}

		node := current.node()
		if node.prefixLen > 0 {
			prefixLen := current.matchDeep(key, depth)
			if prefixLen == 0 {
				return
			} else if depth+prefixLen == len(key) {
				t.forEach(current, callback)
				return
			}
			depth += node.prefixLen
		}

		// Find a child to recursive to
		next := current.findChild(key.charAt(depth))
		if *next == nil {
			return
		}
		current = *next
		depth++
	}
}

// Iterator pattern
func (t *tree) Iterator() Iterator {

	ti := &iterator{
		version:    t.version,
		tree:       t,
		nextNode:   t.root,
		depthLevel: 0,
		depth:      []*iteratorLevel{&iteratorLevel{t.root, 0}}}
	return ti
}

func (ti *iterator) checkConcurrentModification() error {
	if ti.version == ti.tree.version {
		return nil
	}

	return errors.New("Concurrent modification has been detected")
}

func (ti *iterator) HasNext() bool {
	return ti != nil && ti.nextNode != nil
}

func (ti *iterator) Next() (Node, error) {
	if !ti.HasNext() {
		return nil, errors.New("There are no more nodes in the tree")
	}

	err := ti.checkConcurrentModification()
	if err != nil {
		return nil, err
	}

	cur := ti.nextNode
	ti.next()
	return cur, nil
}

func (ti *iterator) next() {
	for {
		var otherNode *artNode = nil
		otherChildIdx := -1

		nextNode := ti.depth[ti.depthLevel].node
		childIdx := ti.depth[ti.depthLevel].childIdx

		switch nextNode.kind {
		case NODE_4:
			node := nextNode.node4()
			i, nodeLimit := childIdx, len(node.children)
			for ; i < nodeLimit; i++ {
				child := node.children[i]
				if child != nil {
					otherChildIdx = i
					otherNode = child
					break
				}
			}

		case NODE_16:
			node := nextNode.node16()
			i, nodeLimit := childIdx, len(node.children)
			for ; i < nodeLimit; i++ {
				child := node.children[i]
				if child != nil {
					otherChildIdx = i
					otherNode = child
					break
				}
			}

		case NODE_48:
			node := nextNode.node48()
			i, nodeLimit := childIdx, len(node.keys)
			for ; i < nodeLimit; i++ {
				idx := node.keys[byte(i)]
				if idx <= 0 {
					continue
				}
				child := node.children[idx-1]
				if child != nil {
					otherChildIdx = i
					otherNode = child
					break
				}
			}

		case NODE_256:
			node := nextNode.node256()
			i, nodeLimit := childIdx, len(node.children)
			for ; i < nodeLimit; i++ {
				child := node.children[i]
				if child != nil {
					otherChildIdx = i
					otherNode = child
					break
				}
			}
		}

		if otherNode == nil {
			if ti.depthLevel > 0 {
				// return to previous level
				ti.depthLevel--
			} else {
				ti.nextNode = nil // done!
				return
			}
		} else {
			// star from the next when we come back from the child node
			ti.depth[ti.depthLevel].childIdx = otherChildIdx + 1
			ti.nextNode = otherNode

			// make sure that w we have enough space for levels
			if ti.depthLevel+1 >= cap(ti.depth) {
				newDepthLevel := make([]*iteratorLevel, ti.depthLevel+2)
				copy(newDepthLevel, ti.depth)
				ti.depth = newDepthLevel
			}

			ti.depthLevel++
			ti.depth[ti.depthLevel] = &iteratorLevel{otherNode, 0}
			return
		}
	}
}