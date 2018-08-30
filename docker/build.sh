#! /bin/bash

#[docker]$ [reg=""] [tag=""] ./build.sh kube-scheduler[,crishim] [--push]

# reg=""
# tag=""

apps=$1
apps=$(echo ${apps//,/ })

push=false
if [ $# -gt 1 ]; then
	shift
	if [ $1 == "--push" ]; then
		push=true
	fi
fi

for app in ${apps[@]}; do
	echo "==> building $app"
	docker build -t $app:test -f $app.Dockerfile ..
	if [ $push == true ]; then
		docker tag $app:test $reg/$app:$tag
		echo "==> pushing $reg/$app:$tag"
		docker push $reg/$app:$tag
	fi
done
