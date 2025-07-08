#!/bin/bash

# after changing this value as part of a rebase, run this script to update deps
release_branch="release-4.20"
# also update this value. You can see what's available via: go list -mod=readonly -m -versions k8s.io/api | sed 's/ /\n/g'
kube_release="v0.33.1"
# these components k8s.io/<item> are versioned for each k8s release
kube_components="api apiextensions-apiserver apimachinery apiserver client-go code-generator component-base kube-aggregator"

echo Updating OpenShift deps to $release_branch

all_mods="$(go list -mod=readonly -m -f '{{ if and (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all)"
declare -A updated_mods

# if something is breaking deps and we pin it, don't do a "go get -u" against it, it will ignore the
# replace directive and upgrade it anyway
replaced_mods=$(go list -mod=readonly -m -f '{{ if and (.Replace) (not .Indirect) (not .Main)}}{{.Path}}{{end}}' all)
declare -A replaced_mods 
for i in $(echo "$replaced_mods"); do 
    replaced_mods["$i"]=1
done


for i in $(echo "$all_mods" | grep '^github.com/openshift/'); do
    echo go get $i@$release_branch
    go get $i@$release_branch
    updated_mods["$i"]=1
done

for i in $kube_components; do
    echo go get k8s.io/$i@$kube_release
    go get k8s.io/$i@$kube_release
    updated_mods["k8s.io/$i"]=1
done

echo
echo Updating all deps
for m in $all_mods; do
    [ -n "${replaced_mods[$m]+1}" ] && echo "WARN: skipping replaced package $m" && continue #it's pinned, leave it alone
    [ -n "${updated_mods[$m]+1}" ] && continue # already updated
    echo go get -u $m
    go get -u $m
    updated_mods["$m"]=1
done

echo
echo go mod tidy
go mod tidy

echo
echo go mod vendor
go mod vendor

echo
echo go mod verify 
go mod verify 
