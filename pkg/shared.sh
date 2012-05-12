build() {
  unset GIT_DIR # this can interefere with "go get"
  msg "Cleaning old build"
  rm -rf $pkgdir/*

  export GOPATH=${srcdir}
  cd ${GOPATH}
  goimport=github.com/nshah/rell
  gitabs=${GOPATH}/src/$goimport

  if [ ! -e ${gitabs} ]; then
    mkdir -p $(dirname ${gitabs})
    cd $(dirname ${gitabs})
    ln -s $srcdir/../../..  $(basename ${gitabs})
  fi

  cd $gitabs

  msg "Getting go dependenices"
  go get -v

  msg "Getting npm dependencies"
  (cd public && npm install)

  bindir=$pkgdir/usr/bin
  mkdir -p $bindir
  binfile=$bindir/$pkgname
  msg "Building"
  go build $goimport -o $binfile

  msg "Copying resources"
  install -d $gitabs/public $pkgdir/usr/share/$pkgname/public
  cp -r $gitabs/public $pkgdir/usr/share/$pkgname
  install -d $gitabs/examples/db/mu $pkgdir/usr/share/$pkgname/examples/mu
  cp -r $gitabs/examples/db/mu $pkgdir/usr/share/$pkgname/examples
  install -d $gitabs/examples/db/old $pkgdir/usr/share/$pkgname/examples/old
  cp -r $gitabs/examples/db/old $pkgdir/usr/share/$pkgname/examples

  msg "Installing rc script"
  install -D $srcdir/../../rc $pkgdir/etc/rc.d/$pkgname

  msg "Creating static resources"
  cd $gitabs/public
  ./node_modules/.bin/browserify -e rell.js > $pkgdir/usr/share/$pkgname/browserify.js
}