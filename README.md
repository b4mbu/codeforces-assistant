## Installation
 - Clone the repository<br>
   `git clone https://github.com/b4mbu/codeforces-assistant.git`
 - Build an application<br>
   `go build -o acf main.go`
 - Add executable file to PATH variable or just move it to */usr/local/bin*<br>
   `mv ./acf /usr/local/bin`

## Commands:
 ### Contest
 Load contest problems in current directory.
 ```shell
 > acf contest [contest number from URL]
 ```
![contest](https://user-images.githubusercontent.com/49525233/230734288-29420dc7-2513-4e3c-87ce-69ee3ebae621.gif)
 
 ### Test
 Test your solution with problem tests.
 
 ```shell
 > acf test [source_file.cpp]
 ```
 ![test](https://user-images.githubusercontent.com/49525233/230734361-aadaaf72-9327-40df-b60f-3c48849e1979.gif)
 
 Use flag `-b [benchmark count]` to get average executing time.
 ```shell
 > acf test [source_file.cpp] -b [benchmark count]
 ```
 ![bench](https://user-images.githubusercontent.com/49525233/230734951-3c507c73-275e-4925-bea2-492b63054a1a.gif)
 *Tip: use `acf test [source_file.cpp] && acf copy [source_file.cpp]` to copy your solution whether it is correct*
## Config:
   Create `acf-config.json` in your user directory.
   Example:
   ```json
   {
       "compiler": "g++",
       "standart": "c++17"
   }
   ```
   
- `compiler` — a name of your C++ compiler. Default is `g++`.
- `standart` — a C++ standart. Default is `c++17`.

