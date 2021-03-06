/*
Copyright 2019 The Skaffold Authors

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

package cache

import (
	"context"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

type mockBuilder struct {
	dependencies []string
}

func (m *mockBuilder) Labels() map[string]string { return nil }

func (m *mockBuilder) Build(ctx context.Context, out io.Writer, tags tag.ImageTags, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	return nil, nil
}

func (m *mockBuilder) DependenciesForArtifact(ctx context.Context, artifact *latest.Artifact) ([]string, error) {
	return m.dependencies, nil
}

func (m *mockBuilder) Prune(ctx context.Context, out io.Writer) error {
	return nil
}

var mockCacheHasher = func(s string) (string, error) {
	return s, nil
}

func TestGetHashForArtifact(t *testing.T) {
	tests := []struct {
		name         string
		dependencies [][]string
		expected     string
	}{
		{
			name: "check dependencies in different orders",
			dependencies: [][]string{
				{"a", "b"},
				{"b", "a"},
			},
			expected: "eb394fd4559b1d9c383f4359667a508a615b82a74e1b160fce539f86ae0842e8",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func(h func(string) (string, error)) { hashFunction = h }(hashFunction)
			hashFunction = mockCacheHasher

			for _, d := range test.dependencies {
				builder := &mockBuilder{dependencies: d}
				actual, err := getHashForArtifact(context.Background(), builder, nil)
				testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, actual)
			}
		})
	}
}
func TestCacheHasher(t *testing.T) {
	tests := []struct {
		name          string
		differentHash bool
		newFilename   string
		update        func(oldFile string, folder *testutil.TempDir)
	}{
		{
			name:          "change filename",
			differentHash: true,
			newFilename:   "newfoo",
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Rename(oldFile, "newfoo")
			},
		},
		{
			name:          "change file contents",
			differentHash: true,
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Write(oldFile, "newcontents")
			},
		},
		{
			name:          "change both",
			differentHash: true,
			newFilename:   "newfoo",
			update: func(oldFile string, folder *testutil.TempDir) {
				folder.Rename(oldFile, "newfoo")
				folder.Write(oldFile, "newcontents")
			},
		},
		{
			name:          "change nothing",
			differentHash: false,
			update:        func(oldFile string, folder *testutil.TempDir) {},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalFile := "foo"
			originalContents := "contents"

			folder, cleanup := testutil.NewTempDir(t)
			defer cleanup()
			folder.Write(originalFile, originalContents)

			path := originalFile
			builder := &mockBuilder{dependencies: []string{folder.Path(originalFile)}}

			oldHash, err := getHashForArtifact(context.Background(), builder, nil)
			if err != nil {
				t.Errorf("error getting hash for artifact: %v", err)
			}

			test.update(originalFile, folder)
			if test.newFilename != "" {
				path = test.newFilename
			}

			builder.dependencies = []string{folder.Path(path)}
			newHash, err := getHashForArtifact(context.Background(), builder, nil)
			if err != nil {
				t.Errorf("error getting hash for artifact: %v", err)
			}

			if test.differentHash && oldHash == newHash {
				t.Fatalf("expected hashes to be different but they were the same:\n oldHash: %s\n newHash: %s", oldHash, newHash)
			}
			if !test.differentHash && oldHash != newHash {
				t.Fatalf("expected hashes to be the same but they were different:\n oldHash: %s\n newHash: %s", oldHash, newHash)
			}
		})
	}
}
