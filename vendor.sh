mkdir vendor || exit
imports=$(go list -f '{{range .Imports}}{{printf "%s\n" .}}{{end}}' ./... | grep -v flargo )
export GOPATH=$PWD/vendor
set -x
for import in $imports; do
	go get "$import" || exit
done
mv vendor/src/* vendor
rm -r vendor/src
