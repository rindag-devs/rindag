with import <nixpkgs> {};
stdenv.mkDerivation {
    name = "rindag-dev-environment";
    buildInputs = [ pkg-config apr.dev aprutil.dev subversion.dev ];
}
