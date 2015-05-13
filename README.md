# Canticle
Canticle is a dependency manager for go. It does revision locking, vanity naming, and vendoring.

## Installing Canticle

Prerequisite: It is assumed that your GO environment is already configured properly (ie. set GOPATH).

```
# Step 1: Open this on your browser (require authentication)
https://github.comcast.com/viper-cog/cant/raw/master/install.sh

# Step 2: Browser will redirect with a token code similar to below
https://github.comcast.com/raw/viper-cog/cant/master/install.sh?token=xxxxxxxxxxxxxxxxxx

# Step 3: Execute the script on your development machine
curl 'https://github.comcast.com/raw/viper-cog/cant/master/install.sh?token=xxxxxxxxxxxxxxxxxx' | sh
```

## Using Canticle

### Create Canticle Defintion File (One Time Step)

On an existing project with 3rd party dependencies, simply execute the following:

```cant save```

Running above step would perform the following:

* Determine all dependencies and their current software version (ie. GIT commitish)
* Create a file called `Canticle` with the above information

Example:

```
[
    {
        "ImportPaths": [
            "code.google.com/p/gcfg",
            "code.google.com/p/gcfg/scanner",
            "code.google.com/p/gcfg/token",
            "code.google.com/p/gcfg/types"
        ],
        "SourcePath": "https://code.google.com/p/gcfg",
        "Root": "code.google.com/p/gcfg",
        "Revision": "c2d3050044d05357eaf6c3547249ba57c5e235cb"
    }
]
```

Tip: To adjust the specific revision of the codebase you want to depend on, simply modify the `Revision` value.

### Using Canticle Definition File

For newly checked out codebase and in order to download the dependencies, simply execute:

```
   cant get     # download all dependencies at their specific revision
   cant build   # build all dependencies
```
