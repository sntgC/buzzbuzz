# buzzbuzz
Quizbowl-style websocket buzzer application built on a golang backend
## Installation
This app is meant to be lightweight and easy to use. Assuming you have administrator privileges, simply download the project files and place them in a folder (you don't need any of the files in the src folder), then run either of the **buzzbuzz_windows** or **buzzbuzz_linux** files. You may have to give the app permission to go through your firewall, which is where the administrator privileges are required. Once the app is running, simply navigate to the ip (found by typing **ipconfig**, for Windows, or **ifconfig**, for Linux, in the command line) of the machine running the executable using a webbrowser from any device on the same network as the host. The default port is :80, but it can be changed using the **addr** flag when running the server.
## Usage
Usage should be pretty intuitive. Once you create a room you will be taken to a control panel, which includes a 5 character code that users can type to join your particular room. In the panel you will also have to option to create and destroy teams, as well as to kick any player from your room. Players can opt to join a single team, or remain teamless. Keep in mind that hosting is only supported on desktop, but users may join from mobile.
### Buzzing
Buzzing is the core of this app. Once a player buzzed, the room is on a "lockdown" during which no player can buzz until the host decides to reset the buzzer. During this "lockdown", the host can award points to the player accordingly depending on whether they got the question right. By default, the "Reward" button on the host panel gives the points to the last person who buzzed and their team, if they are currently in one.
#### *A note on teams:
As the app currently stands, if a player on a team buzzes, their whole team remains on lockdown until the host clears their team. The team reset button is different from the buzzer reset button and is green in color shaped like an arrow going around a dot. You can find it next to each team name on the right part of the control panel. Players on a team currently in lockdown cannot buzz even if they had not previously buzzed.
### Additional Information
This app does not store user sessions or cookies of any sort, so refreshing the page could cause all your progress to be lost. This is planned to be fixed in a future update. 
