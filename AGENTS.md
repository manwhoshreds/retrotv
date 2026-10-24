## Project structure

This projects contains a backend with a web UI and a separate client that will be used on an appliance computer to simulate old school cable television using a provided media library.

- The backend is written in GO and sips with a web UI for configuration
- The client is a C++ / QT app using mpv to play media. 
- When I say frontend, I am refering to the web UI that ships with the backend
- When I say client, I am refering to the actual C++ QT app that is under the client directory


## Styling

- Put frontend CSS styling in external files under backend/web/styles