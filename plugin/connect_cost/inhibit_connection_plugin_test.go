/*
 *  Copyright (c) 2021-2024 Works Applications Co., Ltd.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package connect_cost

import (
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
)

func TestInhibitConnectionPlugin_Edit(t *testing.T) {
	left := int16(0)
	right := int16(0)
	
	// Create mock grammar with 1x1 connection matrix (matching Rust test)
	bytes := buildMockBytes()
	grammar, err := dic.NewGrammar(bytes, 0)
	if err != nil {
		t.Fatalf("Failed to create grammar: %v", err)
	}

	// Create plugin with test pair
	plugin := &InhibitConnectionPlugin{
		inhibitPairs: [][2]int16{{left, right}},
	}

	// Execute edit
	plugin.Edit(grammar)

	// Verify connection is inhibited
	cost := grammar.ConnectCost(left, right)
	if cost != dic.InhibitedConnection {
		t.Errorf("Expected connection cost %d, got %d", dic.InhibitedConnection, cost)
	}
}

func TestInhibitConnectionPlugin_SetUp(t *testing.T) {
	plugin := NewInhibitConnectionPlugin()
	
	// Create test settings matching Rust configuration format
	settings := map[string]any{
		"inhibitPair": [][]int{
			{0, 233},
			{435, 332},
		},
	}

	// Create minimal grammar
	bytes := buildMockBytes()
	grammar, err := dic.NewGrammar(bytes, 0)
	if err != nil {
		t.Fatalf("Failed to create grammar: %v", err)
	}

	// Test setup
	err = plugin.SetUp(settings, "", grammar)
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}

	// Verify pairs were stored correctly
	expected := [][2]int16{{0, 233}, {435, 332}}
	if len(plugin.inhibitPairs) != len(expected) {
		t.Fatalf("Expected %d pairs, got %d", len(expected), len(plugin.inhibitPairs))
	}

	for i, pair := range plugin.inhibitPairs {
		if pair[0] != expected[i][0] || pair[1] != expected[i][1] {
			t.Errorf("Pair %d: expected [%d, %d], got [%d, %d]", 
				i, expected[i][0], expected[i][1], pair[0], pair[1])
		}
	}
}

func TestInhibitConnectionPlugin_GetName(t *testing.T) {
	plugin := NewInhibitConnectionPlugin()
	name := plugin.GetName()
	expected := "InhibitConnectionPlugin"
	if name != expected {
		t.Errorf("Expected name %q, got %q", expected, name)
	}
}

func TestInhibitConnection(t *testing.T) {
	left := int16(0)
	right := int16(0)
	
	// Create mock grammar
	bytes := buildMockBytes()
	grammar, err := dic.NewGrammar(bytes, 0)
	if err != nil {
		t.Fatalf("Failed to create grammar: %v", err)
	}

	// Test inhibit connection function
	InhibitConnection(grammar, left, right)

	// Verify connection is inhibited
	cost := grammar.ConnectCost(left, right)
	if cost != dic.InhibitedConnection {
		t.Errorf("Expected connection cost %d, got %d", dic.InhibitedConnection, cost)
	}
}

// buildMockBytes creates a minimal grammar binary data for testing
// Matches Rust test: fn build_mock_bytes() -> Vec<u8>
func buildMockBytes() []byte {
	var buf []byte
	
	// 0 - pos size (0 POS entries)
	buf = append(buf, 0, 0) // u16 little-endian
	
	// left_id_size = 1
	buf = append(buf, 1, 0) // u16 little-endian
	
	// right_id_size = 1  
	buf = append(buf, 1, 0) // u16 little-endian
	
	// 1x1 connection matrix with 0 cost
	buf = append(buf, 0, 0) // i16 little-endian
	
	return buf
}

// TestRustCompatibility verifies exact compatibility with Rust implementation
func TestRustCompatibility(t *testing.T) {
	// Test that our mock bytes produce the same result as Rust
	bytes := buildMockBytes()
	grammar, err := dic.NewGrammar(bytes, 0)
	if err != nil {
		t.Fatalf("Failed to create grammar: %v", err)
	}

	// Initial cost should be 0 (as set in mock data)
	initialCost := grammar.ConnectCost(0, 0)
	if initialCost != 0 {
		t.Errorf("Expected initial cost 0, got %d", initialCost)
	}

	// After inhibiting, cost should be INHIBITED_CONNECTION
	InhibitConnection(grammar, 0, 0)
	inhibitedCost := grammar.ConnectCost(0, 0)
	if inhibitedCost != dic.InhibitedConnection {
		t.Errorf("Expected inhibited cost %d, got %d", dic.InhibitedConnection, inhibitedCost)
	}

	// Verify INHIBITED_CONNECTION matches Rust i16::MAX
	expectedMax := int16(32767)
	if dic.InhibitedConnection != expectedMax {
		t.Errorf("Expected InhibitedConnection %d, got %d", expectedMax, dic.InhibitedConnection)
	}
}