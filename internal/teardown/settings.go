package teardown

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// errNoGeneratedBase is returned when a structured shared file has no
// recorded generated content, so qsdev's entries cannot be told apart from
// the user's.
var errNoGeneratedBase = errors.New("no recorded generated content in state; qsdev entries cannot be identified, remove them manually")

// hookMatcherKey identifies a hook matcher group inside a settings.json
// hooks event array; a user may add their own hooks to a group qsdev created.
const hookMatcherKey = "matcher"

// stripGeneratedSettings removes from a settings.json document every entry
// qsdev generated, as recorded in base (the generator's output at the last
// write): permission rules, policy fields, hook commands and sandbox entries.
// User additions, user-edited values and keys qsdev never wrote are kept.
func stripGeneratedSettings(current, base []byte) ([]byte, error) {
	if len(bytes.TrimSpace(base)) == 0 {
		return nil, errNoGeneratedBase
	}
	cur, err := decodeJSON(current)
	if err != nil {
		return nil, fmt.Errorf("parsing settings.json: %w", err)
	}
	gen, err := decodeJSON(base)
	if err != nil {
		return nil, fmt.Errorf("parsing recorded generated settings: %w", err)
	}

	stripped, keep := subtractJSON(cur, gen)
	if !keep {
		stripped = map[string]any{}
	}
	out, err := json.MarshalIndent(stripped, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings.json: %w", err)
	}
	return append(out, '\n'), nil
}

func decodeJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// subtractJSON returns cur with everything that also appears in base removed,
// and whether anything is left. Objects are subtracted key by key, arrays
// element by element (order-insensitive), and scalars are removed only when
// unchanged. An array element object sharing its "matcher" with a base
// element is subtracted recursively, keeping the matcher when user content
// remains in the group.
func subtractJSON(cur, base any) (any, bool) {
	if reflect.DeepEqual(cur, base) {
		return nil, false
	}
	switch c := cur.(type) {
	case map[string]any:
		b, ok := base.(map[string]any)
		if !ok {
			return cur, true
		}
		out := make(map[string]any, len(c))
		for k, v := range c {
			bv, inBase := b[k]
			if !inBase {
				out[k] = v
				continue
			}
			if rest, keep := subtractJSON(v, bv); keep {
				out[k] = rest
			}
		}
		return out, len(out) > 0
	case []any:
		b, ok := base.([]any)
		if !ok {
			return cur, true
		}
		out := make([]any, 0, len(c))
		for _, elem := range c {
			if rest, keep := subtractElement(elem, b); keep {
				out = append(out, rest)
			}
		}
		return out, len(out) > 0
	default:
		return cur, true
	}
}

// subtractElement removes elem if base contains an identical element, or
// subtracts the base element sharing its matcher key.
func subtractElement(elem any, base []any) (any, bool) {
	for _, b := range base {
		if reflect.DeepEqual(elem, b) {
			return nil, false
		}
	}
	obj, ok := elem.(map[string]any)
	if !ok {
		return elem, true
	}
	id, hasID := obj[hookMatcherKey]
	if !hasID {
		return elem, true
	}
	for _, b := range base {
		bObj, ok := b.(map[string]any)
		if !ok || !reflect.DeepEqual(bObj[hookMatcherKey], id) {
			continue
		}
		rest, keep := subtractJSON(obj, bObj)
		if !keep {
			return nil, false
		}
		restObj := rest.(map[string]any)
		if _, stillHasHooks := restObj["hooks"]; !stillHasHooks {
			// Only qsdev's hooks lived in this group; nothing of the user's
			// is left to run under the matcher.
			return nil, false
		}
		restObj[hookMatcherKey] = id
		return restObj, true
	}
	return elem, true
}
