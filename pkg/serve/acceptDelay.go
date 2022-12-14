/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package serve

import (
	"time"
)

const (
	maxDelay = 1 * time.Second
)

var AcceptDelay *acceptDelay

func init() {
	AcceptDelay = &acceptDelay{duration: 0}
}

type acceptDelay struct {
	duration time.Duration
}

func (d *acceptDelay) Delay() {
	d.Up()
	d.do()
}

func (d *acceptDelay) Reset() {
	d.duration = 0
}

func (d *acceptDelay) Up() {
	if d.duration == 0 {
		d.duration = 5 * time.Millisecond
		return
	}
	d.duration = 2 * d.duration
	if d.duration > maxDelay {
		d.duration = maxDelay
	}
}

func (d *acceptDelay) do() {
	if d.duration > 0 {
		time.Sleep(d.duration)
	}
}
