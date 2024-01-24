// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package transformers

import (
	"path/filepath"
	"testing"

	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"golang.org/x/tools/go/packages/packagestest"
)

func TestTransformPackages(t *testing.T) {
	// args struct to define input arguments for each test case
	type args struct {
		apiGroupSuffix          string
		resolverFilePattern     string
		ignorePackageLoadErrors bool
		patterns                []string
		inputFilePath           string
	}

	// want struct to define the expected outcome for each test case
	type want struct {
		err             error
		transformedPath string
	}

	// testCase struct for defining test cases
	type testCase struct {
		reason string
		args   args
		want   want
	}

	cases := map[string]testCase{
		"SuccessfulTransformation": {
			reason: "Transformation of the source file that has been generated by crossplane-tool's angryjet succeeds with the expected transformed file.",
			args: args{
				apiGroupSuffix:          "aws.upbound.io",
				resolverFilePattern:     "zz_generated.resolvers.go",
				inputFilePath:           "testdata/SuccessfulTransformation.go.txt",
				ignorePackageLoadErrors: true,
				patterns:                []string{"./testdata"},
			},
			want: want{
				transformedPath: "testdata/SuccessfulTransformation.transformed.go.txt",
			},
		},
		"TransformationIdempotency": {
			reason: "The applied transformation is idempotent, i.e., applying the transformer on an already transformed file does not change the transformed file.",
			args: args{
				apiGroupSuffix:          "aws.upbound.io",
				resolverFilePattern:     "zz_generated.resolvers.go",
				inputFilePath:           "testdata/SuccessfulTransformation.transformed.go.txt",
				ignorePackageLoadErrors: true,
				patterns:                []string{"./testdata"},
			},
			want: want{
				transformedPath: "testdata/SuccessfulTransformation.transformed.go.txt",
			},
		},
		// Other test cases
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inputFileContents := readFile(t, afero.NewOsFs(), tc.args.inputFilePath, tc.reason)
			exported := packagestest.Export(t, packagestest.Modules, []packagestest.Module{{
				Name: "fake",
				Files: map[string]interface{}{
					filepath.Join("testdata", tc.args.resolverFilePattern): inputFileContents,
				}}})
			defer exported.Cleanup()
			exported.Config.Mode = defaultLoadMode
			memFS := afero.NewMemMapFs()
			transformedFilePath := filepath.Join(exported.Temp(), "fake", "testdata", tc.args.resolverFilePattern)
			writeFile(t, memFS, transformedFilePath, []byte(inputFileContents), tc.reason)

			r := NewResolver(memFS, tc.args.apiGroupSuffix, tc.args.ignorePackageLoadErrors, nil, WithLoaderConfig(exported.Config))
			err := r.TransformPackages("zz_generated.resolvers.go", tc.args.patterns...)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nResolver.TransformPackages(...): -wantErr, +gotErr:\n%s", tc.reason, diff)
			}
			if tc.want.err != nil {
				return
			}
			if diff := cmp.Diff(readFile(t, afero.NewOsFs(), tc.want.transformedPath, tc.reason),
				readFile(t, memFS, transformedFilePath, tc.reason)); diff != "" {
				t.Errorf("\n%s\nResolver.TransformPackages(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func readFile(t *testing.T, fs afero.Fs, filePath string, reason string) string {
	buff, err := afero.ReadFile(fs, filePath)
	if err != nil {
		t.Fatalf("\n%s\n: Failed to write the test artifact to the path %s: %v", reason, filePath, err)
	}
	return string(buff)
}

func writeFile(t *testing.T, fs afero.Fs, filePath string, buff []byte, reason string) string {
	err := afero.WriteFile(fs, filePath, buff, 0o600)
	if err != nil {
		t.Fatalf("\n%s\n: Failed to load the test artifact from the path %s: %v", reason, filePath, err)
	}
	return string(buff)
}
