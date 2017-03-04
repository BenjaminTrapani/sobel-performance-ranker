# Install
go install github.com/BenjaminTrapani/sobel-performance-ranker

# Dependencies
Requires diffimg available via ppa dhor/myway
sudo add-apt-repository ppa:dhor/myway
sudo apt-get update; sudo apt-get install diffimg

# Usage notes
Directory layout must be as follows:
-husky_id
--trial 1
---known-image
----output_image
----stdout.txt
----stderr.txt
---unknown-image
----output_image.ppm
----stdout.txt
----stderr.txt
--trial (2...n) same as trial 1
-husky_id_2
....
-husky_id_n

See sobel-performance-ranker -h for flags and descriptions. An example 
input 'test-data' can be run using the 'test-command.sh' bash script.
