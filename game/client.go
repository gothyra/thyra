package game

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"strconv"

)

type Client struct {
	Conn     net.Conn
	Nickname string
	Player   Player
	Ch       chan string
}

func NewClient(c net.Conn, player Player) Client {
	return Client{
		Conn:     c,
		Nickname: player.Nickname,
		Player:   player,
		Ch:       make(chan string),
	}
}

func (c Client) WriteToUser(msg string) {
	io.WriteString(c.Conn, msg)
}

func (c Client) WriteLineToUser(msg string) {
	io.WriteString(c.Conn, msg+"\n\r")
}

func (c Client) ReadLinesInto(ch chan<- string, server *Server) {
	bufc := bufio.NewReader(c.Conn)
	place , _ := server.GetRoom(c.Player.Position)
	xmlStraps , _ := place.GetCubes(c.Player.Position)
	
	s := populate_s(xmlStraps)

	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
			break
		}

		userLine := strings.TrimSpace(line)

		if userLine == "" {
			continue
		}
		lineParts := strings.SplitN(userLine, " ", 2)

		var command, commandText string
		if len(lineParts) > 0 {
			command = lineParts[0]
		}
		if len(lineParts) > 1 {
			commandText = lineParts[1]
		}

		log.Printf("Command by %s: %s  -  %s", c.Player.Nickname, command, commandText)

		switch command {
		case "look":
			fallthrough
		case "watch":
		case "map":
			log.Printf(c.Player.Position)
			updateMap(xmlStraps,c,c.Player.Position,s)
		case "go":
		case "exits":
			log.Printf(c.Player.Position)
						
		printExits(s,c,c.Player.Position)
		case "e":
			fallthrough
		case "east":
		newpos :=findExits(s,c,c.Player.Position)[0]
		if newpos >0{
		c.Player.Position = strconv.Itoa(newpos)
		updateMap(xmlStraps,c,c.Player.Position,s)
		findExits(s,c,c.Player.Position)
		printExits(s,c,c.Player.Position)
		}else {
		c.WriteToUser("You can't go that way\n")
		}
		case "w":
			fallthrough
		case "west":
		newpos :=findExits(s,c,c.Player.Position)[1]
		if newpos >0{
		c.Player.Position = strconv.Itoa(newpos)
		updateMap(xmlStraps,c,c.Player.Position,s)
		findExits(s,c,c.Player.Position)
		printExits(s,c,c.Player.Position)
		}else {
		c.WriteToUser("You can't go that way\n")
		}
		case "n":
			fallthrough
		case "north":
		newpos :=findExits(s,c,c.Player.Position)[2]
		if newpos >0{
		c.Player.Position = strconv.Itoa(newpos)
		updateMap(xmlStraps,c,c.Player.Position,s)
		findExits(s,c,c.Player.Position)
		printExits(s,c,c.Player.Position)
		}else {
		c.WriteToUser("You can't go that way\n")
		}
		case "s":
			fallthrough
		case "south":
		
		newpos :=findExits(s,c,c.Player.Position)[3]
		if newpos >0{
		c.Player.Position = strconv.Itoa(newpos)
		updateMap(xmlStraps,c,c.Player.Position,s)
		findExits(s,c,c.Player.Position)
		printExits(s,c,c.Player.Position)
		}else {
		c.WriteToUser("You can't go that way\n")
		}
		case "say":
			// TODO: implement channel wide communication
			io.WriteString(c.Conn, "\033[1F\033[K") // up one line so we overwrite the say command typed with the result
			ch <- fmt.Sprintf("%s: %s", c.Player.Gamename, commandText)
		case "quit":
		server.OnExit(c)
			c.Conn.Close()
		case "online":
 			for _, nickname := range server.OnlinePlayers() {
 				c.WriteToUser(nickname + "\n")
 			}
			default:
		
					continue
		
			c.WriteToUser("\033[1F\033[K")
		}
	}
}

func (c Client) WriteLinesFrom(ch <-chan string) {
	for msg := range ch {
		_, err := io.WriteString(c.Conn, msg)
		if err != nil {
			return
		}
	}
}

func updateMap(xmlStraps []XMLCube,c Client,pos string,s [][]int) { //Print ton xarti
	
		intpos ,_ :=strconv.Atoi(pos)
 for x := 0; x< len(s); x++ {
	 c.WriteToUser("|") 
   
	  for y := 0; y< len(s); y++ {
		  
        if(s[y][x] != 0 && s[y][x] != intpos){
      c.WriteToUser("___|") 
    } else if (s[y][x] == intpos ){
	
	   c.WriteToUser("_*_|") 
	}else {
c.WriteToUser("_X_|")

	}
	 
}
    c.WriteToUser("\n") 
}
}

func findExits(s [][]int,c Client,pos string) [4] int {  //Briskei ta exits analoga me to position tou user  ,default possition otan kanei register einai 1
	intpos ,_ :=strconv.Atoi(pos)							//to exw balei etsi prwsorina.
	 exitarr :=  [4]int{0 , 0, 0, 0 }						// 2d array s   ,einai o xartis.
for x := 0; x < len(s); x++ {  
	  for y := 0; y < len(s); y++ {							// Kanei return ena array [4]int me ta exits se morfi  {EAST , WEST , NORTH , SOUTH}
		if(s[x][y] == intpos){								//P.X an kanei return  {50,0,40,0} simainei oti apo to possition pou eisai exei exits EAST kai NORTH
																// EAST se paei sto cube me ID 50 kai NORTH se paei sto cube me ID 40
		fmt.Printf("Possition in array :x:%d,y:%d\n" ,x,y) 
		if(x < len(s)-1 && s[x+1][y] > 0){
		exitarr[0] = s[x+1][y]
		}
		if(x >0 && s[x-1][y] >0){
		exitarr[1] = s[x-1][y]
		}
		if(y>0 && s[x][y-1] >0){
		exitarr[2] = s[x][y-1]
		}
		if(y < len(s)-1 && s[x][y+1] >0){
		exitarr[3] = s[x][y+1]
		}	
		}
		
}
}
c.WriteToUser("\n")
fmt.Printf("%v\n",exitarr)
return exitarr
}

func printExits(s [][]int,c Client,pos string) {  //akribws idio func me to findExits , apla edw kanw print ston user.
	intpos ,_ :=strconv.Atoi(pos)
c.WriteToUser("Exits  : ")
for x := 0; x < len(s); x++ {  
	  for y := 0; y < len(s); y++ {
		if(s[x][y] == intpos){			
		fmt.Printf("Possition in array :x:%d,y:%d" ,x,y) 
		if(x < len(s)-1 && s[x+1][y] > 0){
		c.WriteToUser("East ")
		}
		if(x >0 && s[x-1][y] >0){
		c.WriteToUser("West ")
		}
		if(y>0 && s[x][y-1] >0){
		c.WriteToUser("North ")
		}
		if(y < len(s)-1 && s[x][y+1] >0){
		c.WriteToUser("South ")
		}	
		}
		
}
}
c.WriteToUser("\n")

}

func populate_s(xmlStraps []XMLCube) [][]int {  // Edw gemizw ton 2d array s , pou einai o xartis.
													//to xmlStraps einai ena 1d array me ola ta cubes pou pernei apo to XML,
	biggest :=0											// xmlStraps.ID , xmlStraps.POSX kai xmlStraps.POSY ,
	 biggestx, _ := strconv.Atoi(xmlStraps[0].POSX)		// P.X an to 1o cube exei ID=1  POSX=10 POSY=5 
	for v := 0;v< len(xmlStraps);v++ {					// tote tha balei tin timi sto 2d array s[10][5] = 1
		posx,_ := strconv.Atoi(xmlStraps[v].POSX)
		if posx > biggestx {
			biggestx = posx
		}
	}
	fmt.Println("The biggest X is ", biggestx)   // briskw to megalutero POSX apo to xml  xmlStraps.POSX
	

	biggesty, _ := strconv.Atoi(xmlStraps[0].POSY)
	for v := 0;v< len(xmlStraps);v++ {		
		posy,_ := strconv.Atoi(xmlStraps[v].POSY)
		if posy > biggesty {
			biggesty = posy
		}
	}
	fmt.Println("The biggest Y is ", biggesty)  // briskw to megalutero POSY xmlStraps.POSY
	
	
	//Epeidi eixa bug me indexes kai tetoia, edw checkarw pio einai to megalutera apo to POSX kai POSY 
	// PX an to megalutero POSX einai 10 kai to POSY einai 15 , tha ftiaksw enan 2d array 15-15
	//An ftiaksw pinaka 10-15 douleuei swsta alla sto updateMap() exw bug otan ta for pane mexri to len(s) . To opoio tha einai 15 . kai an ftiaksw
	// array 10-15 tote tha crasharei. :) - Tha checkaroume pios einai o swstos tropos na emfanizoume ton xarti. isws na to exw kanei lathos.
	if biggestx > biggesty{
	biggest = biggestx
	}
	if biggestx < biggesty{
	biggest = biggesty
	}else{
		biggest = biggestx
	}
   s := make([][]int,0)

  for i := 0; i <= biggest; i ++{

      tmp := make([]int, 0)
      for j := 0; j <= biggest; j ++{
            tmp = append(tmp, 0)
      }
    s = append(s, tmp)

  } 


fmt.Printf("Len(s) : %d\n",len(s))   



		//Edw briskw ola ta cubes kai bazw tin timi sto 2d array.
		//P.X sto xmlStraps[0]
		  //   <cube id="1"  posx="0" posy="0" />
		  // tha paei sto 2d array kai tha balei  s[0][0]=1
		for z:= 0; z < len(xmlStraps);z++{
		   posx, _ :=strconv.Atoi(xmlStraps[z].POSX)
			posy, _ :=strconv.Atoi(xmlStraps[z].POSY)
			
			if(xmlStraps[z].ID != "" ){
				
			id , _ := strconv.Atoi(xmlStraps[z].ID)
	

			s[posx][posy] = id
		}
	fmt.Printf("Z=%d  --> s[%d,%d] = %d \n", z,posx,posy,s[posx][posy])
	}
        	return s
}

