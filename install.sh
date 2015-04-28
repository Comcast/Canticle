cantdir="$GOPATH/src/github.comcast.com/viper-cog/cant"
vcstools="golang.org/x/tools/go/vcs"
go get $vcstools
mkdir -p $cantdir
git clone git@github.comcast.com:viper-cog/cant.git $cantdir
cd $cantdir
go build
./cant build -l
cp cant "$GOPATH/bin"
