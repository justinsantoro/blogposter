#!/usr/bin/bash


# check if /contentrepo/.git exists

cd /contentrepo
git init
git remote add origin https://github.com/sarahlehman/smalltownkitten
git fetch --depth=1
git reset origin/master
git sparse-checkout init --cone
git restore .
git sparse-checkout set content data
#git checkout other branch?

blogposter $@