# Canticle

Canticle is a dependency manager for go. But, Canticle is also so much
more. It's for version locking libraries, single projects, and entire
continuously-released, microservice platforms. It's also for vendoring
internally, _without_ import path rewriting.

# Installing Canticle

Prerequisite: It is assumed that your GO environment is already configured properly (ie. set GOPATH).

```
# Step 1: Open this on your browser (require authentication)
https://github.comcast.com/viper-cog/cant/raw/master/install.sh

# Step 2: Browser will redirect with a token code similar to below
https://github.comcast.com/raw/viper-cog/cant/master/install.sh?token=xxxxxxxxxxxxxxxxxx

# Step 3: Execute the script on your development machine
$ curl 'https://github.comcast.com/raw/viper-cog/cant/master/install.sh?token=xxxxxxxxxxxxxxxxxx' | sh
```

# Using Canticle
## On a Library

As a library maintainer you may need to save a certain revision, tag,
or branch of a project you rely on.

1.  Checkout the version of your dependent repos you rely on and run:

2.  Save those versions, at the root of your project run:
	```
	cant save --no-sources
	```
	
	If you want to save a branch (say release/1.2.0) just run:
	```
	cant save -b --no-sources
	```

3.  Commit the `Canticle` file.

## On a Single Project

If you maintain a single project with many dependencies for your
company it is: probably closed source, probably needs vendoring, and
probably needs revision locks. For this we:

1.  Grab the dependency:
	```
	cant vendor <external_repo>
	```

2.  Import a copy into your internal repos. In git we might do this
like so:
	```
	git remote rm origin
	git remote add origin <internal_repo>
	git push -u origin --all
	```

3.  Save the project deps:
	```
	cd $GOHOME/<your_project>
	cant save
	```
	If conflicts exists (multiple specified vcs sources or revisions) you
	will be prompted for resolution. To prefer whats on disk now simply do
	`cant save -ondisk`.

4.  Check in the resulting `Canticle` file.

5.  Work with the project like normal. New developers can simply:
	```
	cd $GOHOME/<your_project>
	cant get
	```
	and all of your repos will be pulled at the correct revision, from the
	correct source (your internal repo).

6.  Update a dependency. Use its vcs to pull in updates, then checkout
	the revision you want. Goto 3.


## On a Workspace (Many Related Projects)

If you maintain a system of microservices (many go projects, possibly
with overlapping dependencies) then you probably want a workspace for
those projects. A single project with several independently version
controlled libraries may also justify a workspace.

### Creating

1.  Create your workspace:
	We will save all dependencies for the workspace at the `src/` level of
	our `$GOHOME`. This will then be checked in. Generally a workspace
	will be a VCS project containing a file structure like this:
	```
	src/
	src/Canticle
	```

2.  Layout your workspaces `$GOHOME` with all the projects you care
    about.

3.  Checkout the revision or branch you want for each project in the
    workspace.
	
4.  Save the projects dependencies, exclude libraries who are imported
	by your project:
	```
	cd $GOHOME/src	
	cant save -b -exclude golang.org # We add -b to save branches/tags instead of revisions
	git add Canticle
	git commit -m "Initial commit"
	git push
	```

5.  Create an alternative view of the world. Since your workspace is
	itself under version control, its possible to create many "views"
	of the world. For example we could check in a `Canticle` file
	containing all development versions instead of master:
	```
	for project in $PROJECTS; do pushd $project && git checkout tip && popd; done;
	cd $GOHOME/src	
	cant save -b -exclude golang.org
	git checkout -b "tip"
	git commit -am "Canticle file with just tip development branches"
	git push -u origin tip
	```	

### Using 

1.  Clone the workspace project. Set your gopath to its root `export
    GOHOME=$(pwd)`.

2.  Grab all the projects in the workspace:
	```
	cd $GOHOME/src
	cant get
	```

3.  Develop and commit like normal against a projects VCS.
	Check you didn't break the world:
	```
	cd $GOHOME/src
	go test ./...
	go build ./...
	```

4.  Grab any updates for the workspace:
	```
	cd $GOHOME/src
	git pull --ff-only # Grab any workspace changes
	cant get -u # Update any changed projects in the workspace
	```

5.  Save any updates to the workspace:
	```
	cd $GOHOME/src
	cant save -b
	git commit -m 'Updated workspace deps'
	git push
	```

6.  Checkout another view of the world. For this we checkout the
    alternative branch at the VCS level and grab get the repos again:
	```
	cd $GOHOME/src
	git checkout tip
	cant get -u
	```

# Generating Version Information

Having all this version information is useless if you can't see it at
runtime. Luckily Canticle has your back.

1.  `cd` to your main package directory, e.g. for this project `cant`.
	
2.  Generate the version info:
	```
	cant genversion
	```
	
3.  Check in the created `buildinfo/info.go` file. DO NOT check in
    `buildinfo.go`. I recommend adding this to your `.gitignore.`

4.  Import `buildinfo` and use it. Before building run `cant
    genversion` everytime.

# Understanding Canticle

## Running `cant save`

## Running `cant get`

## Running `cant vendor`

