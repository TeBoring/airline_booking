// Unbounded buffer, where underlying values are arbitrary values
package storageimpl

// Linked list element
type HlEle struct {
	val string
	next *HlEle
	before *HlEle
}

type HashList struct {
	head *HlEle         // Oldest element
	tail *HlEle         // Most recently inserted element
	listMap map[string](*HlEle)
	cnt int
}

func NewHashList() *HashList {
	hl := new(HashList)
	hl.listMap = make(map[string](*HlEle))
	return hl
}
// Insert
// - add new elements to the tail
// - create map entry to point to it
func (hl *HashList) Insert(val string) {
	ele := &HlEle{val : val}
	if hl.head == nil {
		// Inserting into empty list
		hl.head = ele
	} else {
		hl.tail.next = ele
		ele.before = hl.tail
	}
	hl.tail = ele
	hl.listMap[val] = ele
	hl.cnt = hl.cnt + 1
}

// Remove
// - use map to find target (O(1))
func (hl *HashList) Remove(val string) {
	e, ok := hl.listMap[val]
	if ok == true {
		if e.before != nil {
			e.before.next = e.next
		}
		if e.next != nil {
			e.next.before = e.before
		}
		if hl.head == e {
			hl.head = e.next
		}
		if hl.tail == e {
			hl.tail = e.before
		}
		delete(hl.listMap, val)
		hl.cnt = hl.cnt - 1;
	}
}

func (hl *HashList) Exist(val string) bool {
	_, ok := hl.listMap[val]
	return ok
}

func (hl *HashList) List() []string {
	returnList := make([]string, hl.cnt)
	lp := hl.head
	for i := 0; i < hl.cnt; i++ {
		returnList[i] = lp.val
		lp = lp.next
	}
	return returnList
}
