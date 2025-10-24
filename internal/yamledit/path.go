package yamledit

import "strings"

var PathCleaner = strings.NewReplacer(".", "_", " ", "_", "@", "")
