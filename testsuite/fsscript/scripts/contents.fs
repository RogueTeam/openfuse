= Simple file operations
mkdir secrets 700
ls secrets EMPTY
file secrets/readme.md 644 Hello world
cat secrets/readme.md Hello world
perm secrets 700
type secrets dir
perm secrets/readme.md 644
type secrets/readme.md file
ls secrets readme.md

= Chmod a directory
mkdir secrets 700
chmod secrets 777
perm secrets 777
type secrets dir
ls secrets EMPTY

= Empty file
file empty.md 644 EMPTY
cat empty.md EMPTY
size empty.md 0
