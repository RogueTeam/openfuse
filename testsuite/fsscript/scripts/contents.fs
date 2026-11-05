= Empty file
file empty.md 644 EMPTY
cat empty.md EMPTY
size empty.md 0

= Exact size
file 6bytes.txt 644 123456
size 6bytes.txt 6
cat 6bytes.txt 123456

= Short appends
file readme 644 EMPTY
append readme Hello
append readme World
append readme !!
cat readme HelloWorld!!

= Large multi-chunk file
randomfile readme 204800
size readme 204800

= Write past the end
file readme 644 abc
writeat readme 10 Z
writeat readme 3 0000000
cat readme abc0000000Z
size readme 11
readat readme 0 3 abc
readat readme 3 7 0000000
readat readme 10 1 Z

= Overwrite inside
file readme 644 abcdef
writeat readme 2 XY
size readme 6
cat readme abXYef

= Overwrite past the appends
file readme 644 abcdef
writeat readme 4 XYZW
size readme 8
cat readme abcdXYZW

= Truncate down and up
file readme 644 abcdef
truncate readme 3
cat readme abc
size readme 3
truncate readme 6
size readme 6

= Rewrite file with shorter content
file readme 644 abcdef
file readme 644 abc
cat readme abc
size readme 3
truncate readme 3
size readme 3
cat readme abc

= Truncate to zero, then append
file readme 644 abcdef
truncate readme 0
append readme hello world
cat readme hello world

= Chmod a directory
mkdir secrets 700
chmod secrets 777
perm secrets 777
type secrets dir
ls secrets EMPTY

= Create directory and child file
mkdir secrets 700
ls secrets EMPTY
file secrets/readme.md 644 Hello world
cat secrets/readme.md Hello world
perm secrets 700
type secrets dir
perm secrets/readme.md 644
type secrets/readme.md file
ls secrets readme.md
