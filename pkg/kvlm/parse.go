package kvlm

// Key-value List with Message
type Kvlm struct {
	Message []byte
	Okv     OrderedKV
}

func New() *Kvlm {
	return &Kvlm{
		Message: []byte{},
		Okv:     NewOrderedKV(),
	}
}

// Parse kvlm as byte array into kvlm struct
func Parse(raw []byte, start int, msg *Kvlm) error {
	// This function is recursive: it reads a key/value pair, then calls
	// itself back with a pointer to the next kv pair.  So we first need to know
	// where we are: at a keyword, or already in the MessageQ

	// We search for the next space and new line
	spaceIndex := find(raw, ' ', start) + start
	newlineIndex := find(raw, '\n', start) + start

	// If a space appears before a new line, we have a keyword. Otherwise,
	// it's the final Message, which we just read to the end of the file

	// Base case
	// =========
	// If newline appears first (or there is no space at all, in which case
	// find returns -1), we assume a blank line. A blank line means that the
	// remainder of the data is the Message. We store it in the kvlm and return.
	if (spaceIndex < 0) || (newlineIndex < spaceIndex) {
		msg.Message = raw[start+1:]
		return nil
	}

	// Recursive case
	// ==============
	// We read a key-value pair and recurse for the next

	// Read the key
	key := string(raw[start:spaceIndex])

	// Then, find the end of the value. Continuation lines begin
	// with a space, so we loop until we find a \n not followed
	// by a space (because values can be multi-line)
	end := start
	for {
		end += find(raw, '\n', end+1)
		if end >= len(raw) || raw[end+1] != ' ' {
			break
		}
	}

	end += 1

	// Then we can get the value
	val := raw[spaceIndex+1 : end]

	// And put the value in the map
	if mapVal, ok := msg.Okv.Get(key); ok {
		msg.Okv.Set(key, append(mapVal, val...))
	} else {
		msg.Okv.Set(key, val)
	}

	// Finally, recurse over the other values
	return Parse(raw, end+1, msg)
}

func find(raw []byte, char byte, start int) int {
	for idx, val := range raw[start:] {
		if val == char {
			return idx
		}
	}
	return -1
}
