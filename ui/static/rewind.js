
// Show info
var showStartTimeEpoch; // Number: hours since epoch that current show started (used for API)
var showStartTime; // Date: Start time of show
var showEndTime; // Date: End time of show
var showDurationMs; // Number: Duration of entire show (not recording) in milliseconds.

// TimeMachine audio info
var recordingStartTime; // Date: Start time of recording  (in case recorder was started late)
var recordingPlaybackTime; // Date: Current time of recording playback as date obj
var playbackStartTime; // Date: time in show that current seek playback started (with the file offset requested)

var pastUnplayablePercent; // Number (0-100): Percentage of audio not available at beginning of show (recorder restarted)

var seekingTimeout; // Timeout function so that isDragging doesn't spam timemachine with constant different offset requests.
var isDragging; // User is dragging the seek bar around.

var requestAPI = function(endpoint, readyfunction) {
  var xmlhttp = new XMLHttpRequest();
  xmlhttp.open('GET', endpoint, true);
  xmlhttp.onreadystatechange = function() {
    if (xmlhttp.readyState == 4 && xmlhttp.status == 200) {
      var obj = JSON.parse(xmlhttp.responseText);
      readyfunction(obj)
    } else {
      readyfunction()
    }
  };
  xmlhttp.send(null);
}

var requestShowInfo = function() {
  requestAPI('https://ury.org.uk/api/v2/timeslot/currentandnext?api_key=' + window.APIKey, (obj) => {
    if (!obj.payload) {
      console.log("RIP! API failed.")
      return
    }
    var payload = obj.payload;
    var currentShow = payload.current;

    // If the current show has no start time (jukebox/off air), allow seek to beginning of hour.
    if (!currentShow.start_time) {
      var start_time = new Date();
      start_time.setHours(start_time.getHours())
      start_time.setMinutes(0);
      start_time.setSeconds(0);
      currentShow.start_time = start_time.getTime() / 1000;
    }
    if (isNaN(currentShow.end_time)) {
      var end_time = new Date();
      end_time.setHours(end_time.getHours() + 1)
      end_time.setMinutes(0);
      end_time.setSeconds(0);
      currentShow.end_time = end_time.getTime() / 1000;
    }

    showStartTimeEpoch = Math.round(currentShow.start_time / (60*60));

    showStartTime = new Date(currentShow.start_time * 1000)
    showEndTime = new Date(currentShow.end_time * 1000)
    showDurationMs = (showEndTime.getTime() - showStartTime.getTime())

    // Display the show info
    console.log(currentShow)
    document.title = currentShow.title + " - URY Rewind"
    audio.setAttribute("title", currentShow.title + " - URY Rewind")
    $(".show-name").text(currentShow.title)
    $(".show-date").text(showStartTime.toDateString())
    $(".show-starttime").text(showStartTime.toLocaleTimeString())
    $(".show-endtime").text(showEndTime.toLocaleTimeString())
    $(".show-image").attr("src", "https://ury.org.uk" + currentShow.photo);

    requestShowTM();
  });
}

// Get recording data about the loaded show from Time Machine
var requestShowTM = function() {
  requestAPI("tm/v1/show/" + showStartTimeEpoch, (obj) => {

    if (obj.Available) {
      if (obj.Playable > 0) {
        var playableSecs = obj.Playable
        console.log("PlayableSeconds", playableSecs)
        var now = new Date();
        recordingStartTime = new Date(Math.min(now.getTime(),showEndTime.getTime()) - playableSecs*1000)
        if (recordingStartTime.getTime() < showStartTime.getTime()) {
          // If timemachine is telling us it has more audio than is sensitible, let's correct it.
          recordingStartTime = showStartTime
        }
        console.log("Recording start time", recordingStartTime.toLocaleString())

        // Now we need to work out when the recording scrubbing bar should start, if we don't have all of the audio since the start of show. (Recorder started late etc)
        var pastUnplayableDurationMs = Math.max(0, recordingStartTime.getTime() - showStartTime.getTime())
        console.log("PastUnplayable Duration", pastUnplayableDurationMs)
        // The following percentage of the show is unplayable at the beginning,
        pastUnplayablePercent = (pastUnplayableDurationMs / showDurationMs) * 100
        // this remains static over the whole show, so set the CSS now.
        $("#audio-progress").css(
          {
            'left': pastUnplayablePercent + "%"
          }
        );

        // Get the player to load the start of the show, paused.
        seekToProgress(0, false)
      }

    }
  });
}
// Start it all off!
requestShowInfo()

var showElapsedPercent = function() {
  var now = new Date();

  return ((Math.min(now.getTime(),showEndTime.getTime()) - showStartTime.getTime()) / showDurationMs) * 100
}

var player = document.getElementById('audio');
player.addEventListener("timeupdate", function() {

  if (playbackStartTime && !isDragging) {
    $(".playback-time").text(playerTime().toLocaleTimeString())


  var percent_through_show = showElapsedPercent()

  $("#audio-progress").css(
    {
      'width': percent_through_show - pastUnplayablePercent + "%"
    }
  );

  var now = new Date();
  var percent = (playerTime().getTime() - recordingStartTime.getTime()) / (Math.min(now.getTime(),showEndTime.getTime()) - recordingStartTime.getTime())*100
    $('#audio-progress-bar').css({
      'width': percent + "%"
    });
    $('#draggable-point').css({
      'left': percent + "%"
    });
  }
});

var playerTime = function(tempPlaybackOffsetS) {
  var currentPlaybackTime
  if (tempPlaybackOffsetS) {
    currentPlaybackTime = new Date(recordingStartTime.getTime() + tempPlaybackOffsetS*1000);
  } else {
    if (playbackStartTime) {
      currentPlaybackTime = new Date(playbackStartTime.getTime() + player.currentTime*1000);
    }
  }

  return currentPlaybackTime
}

var seekToProgress = function(offset, startPlaying = true) {
  changePlayState(false)
  isDragging = false;
  offset = Math.round(offset);
  console.log("Seeking to offset:", offset)

  var audio = document.getElementById('audio');

  var source = document.getElementById('audioSource');
  source.src = "tm/v1/show/" + showStartTimeEpoch + "/stream?offset=" + offset

  audio.load(); //call this to just preload the audio without playing

  if (startPlaying) {
    changePlayState(true); //call this to play the song right away
  }

  playbackStartTime = new Date(recordingStartTime.getTime() + offset*1000);

}
var playState;

var pButton = "#show-player-play"; // Play/Pause/Stop button

function changePlayState(state) {
  //Only modify audio player if we're making a change.
  if (state) {
    player.play();
    $(pButton).removeClass("fa-play").removeClass("fa-close").addClass("fa-pause");
  } else {
    player.pause();
    $(pButton).removeClass("fa-pause").removeClass("fa-close").addClass("fa-play");
  }
  playState = state;
};

$(pButton).click(function(e) {
  changePlayState(!playState);
});

$('#audio-progress').click(function(e) {
  var elm = $(this);
  console.log(e.pageX, elm.offset().left, elm.width());
  var xPos = (e.pageX - elm.offset().left) / elm.width();



  console.log(xPos)

  var requested_percent = xPos*100; //*percent_through_show;
  console.log("Requested %", requested_percent)

  var now = new Date();
  var recordingDurationS = (Math.min(now.getTime(),showEndTime.getTime()) - recordingStartTime.getTime())/1000
  offset = requested_percent/100*(recordingDurationS)
  console.log(offset)
  seekToProgress(offset)
})

$('#draggable-point').draggable({
  axis: 'x',
  containment: "#audio-progress"
});

$('#draggable-point').draggable({
  drag: function() {
    isDragging = true;
    var offset = $(this).offset();
    var xPos = (100 * parseFloat($(this).css("left"))) / (parseFloat($(this).parent().css("width")))


    //console.log(xPos)

    var requested_percent = xPos; //*percent_through_show/100
    console.log("Requested percent", requested_percent)

    var now = new Date();
    var recordingDurationS = (Math.min(now.getTime(),showEndTime.getTime()) - recordingStartTime.getTime())/1000
    offset = requested_percent/100*(recordingDurationS)

    //console.log("percent through show", percent_through_show)

    //console.log("handle offset", handle_offset)

    //console.log("Behind by", offset)
    $(".playback-time").text(playerTime(offset).toLocaleTimeString());
    if (seekingTimeout) {
      clearTimeout(seekingTimeout)
    }
    seekingTimeout = setTimeout(() => {seekToProgress(offset)}, 300)
  }
});
