package soldr

import (
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type processedField struct {
	value *protoreflect.Value
	set   bool
}

type cache map[string]*processedField

type fieldProcessor struct {
	cache cache
}

func newFieldProcessor() *fieldProcessor {
	return &fieldProcessor{
		cache: make(cache),
	}
}

func (fp *fieldProcessor) isFieldSet(path string, message proto.Message) bool {
	if message == nil || path == "" || path == "." {
		return false
	}

	msgReflect := message.ProtoReflect()
	descriptor := msgReflect.Descriptor()
	fieldParts := strings.Split(path, ".")
	currentPath := ""

	// traverse the labels and check the fields
	for i, part := range fieldParts {
		// calculate the current path
		if currentPath == "" {
			currentPath = path
		} else {
			currentPath = currentPath + "."
		}

		// check the cache
		cached, ok := fp.cache[currentPath]
		if ok {
			if !cached.set {
				return false
			}

			// if we're at the end and it's set, return true
			if i == len(fieldParts)-1 {
				return true
			}

			// if we're not at the end and there's a nil message, return false
			if cached.value == nil || cached.value.Message() == nil {
				return false
			}

			// set the next values
			msgReflect = cached.value.Message()
			descriptor = msgReflect.Descriptor()
			continue
		}

		// if not set in cache, check the message decriptor for the name
		field := descriptor.Fields().ByName(protoreflect.Name(part))
		if field == nil {
			// check by json name if not found
			field = descriptor.Fields().ByJSONName(part)
		}
		if field == nil {
			// if it's not found, set an empty entry for the label and return false
			fp.cache[currentPath] = &processedField{}
			return false
		}

		// get if the field is set on the message and the value
		set := msgReflect.Has(field)
		fieldValue := msgReflect.Get(field)
		fp.cache[currentPath] = &processedField{
			value: &fieldValue,
			set:   set,
		}

		// if we're done, return whether the field is set
		if i == len(fieldParts)-1 {
			return set
		}

		if fieldValue.Message() == nil {
			return false
		}

		msgReflect = fieldValue.Message()
		descriptor = msgReflect.Descriptor()
	}

	return false
}
