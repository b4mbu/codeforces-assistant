## Installation
 - Clone the repository
   `https://github.com/b4mbu/codeforces-assistant.git`
 - Build an application
   `go build main.go -o acf`
 - Add executable file to PATH variable or just move it to `/usr/local/bin`
   `mv ./acf /usr/local/bin`

## Commands:
 - `contest` -- load contest problems in current directory.
 - `test [source_file.cpp]` -- test your solution with problem tests. Use flag `-b [benchmark count]` to get average executing time.
 - `copy [source_file.cpp]` -- copy your solution to clipboard. 
   *Hint: use `acf test [source_file.cpp] && acf copy [source_file.cpp]` to copy your solution whether it is correct*
## Config:
   ##### Create `acf-config.json` in your user directory.
   ##### Fields:
- `compiler` -- a name of your C++ compiler. Default is `g++`.
- `standart` -- a C++ standart. Default is `c++17`.

