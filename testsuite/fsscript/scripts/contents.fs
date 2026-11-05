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
