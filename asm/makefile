.SUFFIXES:

TARGETS = demo 

all: $(TARGETS)

%: %.o
	ld -o $@ $< -e _start -arch arm64 -platform_version macos 15.0 15.0 -lSystem -L$(shell xcrun --show-sdk-path)/usr/lib

%.o: %.s
	as -o $@ $<

clean:
	rm -f *.o $(TARGETS)

test: $(TARGETS)
	./demo
	echo "demo exit code: $?"
	./demo2
	echo "demo2 exit code: $?"

.PHONY: all clean test
