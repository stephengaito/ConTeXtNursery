// Copyright 2020 PerceptiSys Ltd, (Stephen Gaito)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
  "github.com/diSimplex/ConTeXtNursery/clientConnection"
  "github.com/diSimplex/ConTeXtNursery/interfaces/discovery"
  "math/rand"
  "time"
)

func grimReaper(cc *clientConnection.CC) {
  // if we are not the primary Nursery... don't do anything...
  if ! isPrimary() { return }

  for {
    time.Sleep(time.Duration(rand.Int63n(20)) * time.Second)

    deadNurseries := make([]string, 0)

    cnInfoMap.DoToAllOthers(func (name string, ni discovery.NurseryInfo) {
      replyBytes := cc.GetMessage(ni.Base_Url, "/")
      if len(replyBytes) < 1 {
        // could not reach this Nursery.... so reap it!
        deadNurseries = append(deadNurseries, name)
      }
    })

    cnInfoMap.DeleteAll(deadNurseries)
  }
}