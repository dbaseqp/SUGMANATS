package models

import (
)

func (t UserData) IsValid() bool {
	return t.Name != ""
}