{ runCommand, fetchurl, makeWrapper, patchelf, libffi, mupdf, glibc, stdenv, lib }:

runCommand "pdf-cli-2.0" {
  nativeBuildInputs = [ makeWrapper patchelf ];
  buildInputs = [ libffi mupdf glibc ];
} ''
  mkdir -p $out/bin
  cp ${fetchurl {
    url = "https://github.com/Yujonpradhananga/pdf-cli/releases/download/v.2.0/pdf-cli";
    hash = "sha256-JKQBouz1Qum7IE+8+vgOTi5/j1FovesmE0cOqrlsCjk=";
  }} $out/bin/.pdf-cli-unwrapped
  chmod 755 $out/bin/.pdf-cli-unwrapped
  
  patchelf \
    --set-interpreter ${stdenv.cc.bintools.dynamicLinker} \
    --set-rpath ${lib.makeLibraryPath [ libffi mupdf glibc ]} \
    $out/bin/.pdf-cli-unwrapped
  
  makeWrapper $out/bin/.pdf-cli-unwrapped $out/bin/pdf-cli \
    --set LD_LIBRARY_PATH ${lib.makeLibraryPath [ libffi mupdf glibc ]}
''
