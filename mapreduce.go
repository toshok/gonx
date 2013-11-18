package gonx

import (
	"bufio"
	"io"
	"sync"
)

func handleError(err error) {
	//fmt.Fprintln(os.Stderr, err)
}

// Iterate over given file and map each it's line into Entry record using parser.
// Results will be written into output Entries channel.
func EntryMap(file io.Reader, parser *Parser, output chan Entry) {
	// Input file lines. This channel is unbuffered to publish
	// next line to handle only when previous is taken by mapper.
	var lines = make(chan string)

	// Host thread to spawn new mappers
	var quit = make(chan int)
	go func(topLoad int) {
		// Create semafore channel with capacity equal to the output channel
		// capacity. Use it to control mapper goroutines spawn.
		var sem = make(chan bool, topLoad)
		for i := 0; i < topLoad; i++ {
			// Ready to go!
			sem <- true
		}

		var wg sync.WaitGroup
		for {
			// Wait until semaphore becomes available and run a mapper
			if !<-sem {
				// Stop the host loop if false received from semaphore
				break
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Take next file line to map. Check is channel closed.
				line, ok := <-lines
				// Return immediately if lines channel is closed
				if !ok {
					// Send false to semaphore channel to indicate that job's done
					sem <- false
					return
				}
				entry, err := parser.ParseString(line)
				if err == nil {
					// Write result Entry to the output channel. This will
					// block goroutine runtime until channel is free to
					// accept new item.
					output <- entry
				} else {
					handleError(err)
				}
				// Increment semaphore to allow new mapper workers to spawn
				sem <- true
			}()
		}
		// Wait for all mappers to complete, then send a quit signal
		wg.Wait()
		quit <- 1
	}(cap(output))

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Read next line from the file and feed mapper routines.
		lines <- scanner.Text()
	}
	close(lines)

	if err := scanner.Err(); err != nil {
		handleError(err)
	}

	<-quit
}
