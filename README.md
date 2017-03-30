#Create gif from webm
```
mkdir frames
ffmpeg -i Screencast.webm  -r 5 'frames/frame-%03d.png'
convert -delay 20 -loop 0 frames/*.png screencast.gif
```