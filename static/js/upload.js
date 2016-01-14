function FileAPI (c, t, d, f, tag, url) {

    var fileCount = c,
        fileList = t,
        dropZone = d,
        fileField = f,
        counter_queue = 0,
        counter_uploading = 0,
        counter_completed = 0,
        counter_failed = 0,
        concurrency = 4,
        fileQueue = new Array(),
        preview = null;


    this.init = function () {
        fileField.onchange = this.addFiles;
        dropZone.addEventListener("dragenter",  this.stopProp, false);
        dropZone.addEventListener("dragleave",  this.dragExit, false);
        dropZone.addEventListener("dragover",  this.dragOver, false);
        dropZone.addEventListener("drop",  this.showDroppedFiles, false);
    }

    this.addFiles = function () {
        addFileListItems(this.files);
    }

    function updateFileCount() {
	var box = document.getElementById('fileCount');

        // XXX: Make this less messy
        var text = counter_completed + " of " + counter_queue + " file";
        if (counter_queue != 1){
            text = text + "s";
        }
        text = text + " uploaded";
        if (counter_failed > 0) {
            text = text + ". " + counter_failed + " failed.";
            box.className = "alert alert-danger";
        } else if (counter_completed == counter_queue) {
            text = text + ", all done!";
            box.className = "alert alert-success";

            // Automatic refresh when uploads complete
            location.reload(true);
        }

        if ((counter_completed + counter_failed) != counter_queue) {
            text = text + "...";
            box.className = "alert alert-info";
        }

        fileCount.textContent = text;
	box.style.display = 'block';
    }
    this.showDroppedFiles = function (ev) {
        ev.stopPropagation();
        ev.preventDefault();
        dropZone.style["backgroundColor"] = "#FFFFFF";
        var files = ev.dataTransfer.files;
        addFileListItems(files);
    }

    this.dragOver = function (ev) {
        ev.stopPropagation();
        ev.preventDefault();
        this.style["backgroundColor"] = "#EEEEEE";
    }

    this.dragExit = function (ev) {
        ev.stopPropagation();
        ev.preventDefault();
        dropZone.style["backgroundColor"] = "#FFFFFF";
    }
    this.stopProp = function (ev) {
        ev.stopPropagation();
        ev.preventDefault();
    }

    this.uploadQueue = function (ev) {
        ev.preventDefault();

        // Loop that will wait 100ms between each iteration
        var i = setInterval(function(){
            // Initiate a upload if within the concurrency limit
            if (counter_uploading < concurrency) {
                var item = fileQueue.pop();
                uploadFile(item.file, item.container);
            }

            // Break out of the loop when the queue is empty
            if (fileQueue.length == 0) {
                clearInterval(i);
            }
        }, 100);
    }

    var addFileListItems = function (files) {
        counter_queue += files.length;
        updateFileCount();
        for (var i = 0; i < files.length; i++) {
            showFileInList(files[i])
        }
    }

    var showFileInList = function (file) {
        //var file = ev.target.file;
        if (file) {
            var container = document.createElement("p");
            //container.className = "list-group-item";

            var meta = document.createElement("div");
            meta.className = "row";

            var name = document.createElement("div");
            var strong = document.createElement("strong");
            var nameText = document.createTextNode(file.name);
            strong.appendChild(nameText);
            name.appendChild(strong);
	    name.className = "col-md-8";
            meta.appendChild(name);

            var filesize = getReadableFileSizeString(file.size);
            //var size = document.createElement("div");
            //var sizeText = document.createTextNode(filesize);
            //size.appendChild(sizeText);
	    //size.className = "col-md-2";
            //meta.appendChild(size)

            var speed = document.createElement("div");
            //var mimeText = document.createTextNode(mimetype);
            speed.textContent = "Pending (" + filesize + ")";
            speed.className = "col-md-4 text-right";
            meta.appendChild(speed)

            // Progressbar
            var barcontainer = document.createElement("div");
            barcontainer.className = "row";

            var bar = document.createElement("div");
	    bar.className = "col-md-12";

            var progress = document.createElement("progress");
            progress.max = 100;
            progress.value = 0;
            progress.className = "progress";
            bar.appendChild(progress);

            barcontainer.appendChild(bar);

            container.appendChild(meta)
            container.appendChild(barcontainer)

            fileList.insertBefore(container, fileList.childNodes[0]);
            updateFileCount();
            fileQueue.push({
                file : file,
                container : container
            });
        }
    }

    function roundNumber(num, dec) {
        var result = Math.round(num*Math.pow(10,dec))/Math.pow(10,dec);
        return result;
    }

    function humanizeBytesPerSecond(speed) {
        var unit = "KB/s";
        if (speed >= 1024) {
            unit = "MB/s";
            speed /=1024;
        }
        return (speed.toFixed(1) + unit);
    };


    var uploadFile = function (file, container) {
        if (container && file) {
            counter_uploading += 1;

            var filesize = getReadableFileSizeString(file.size);
            var speed = container.getElementsByTagName("div")[2];
            var bar = container.getElementsByTagName("div")[4];
            var progress = bar.getElementsByTagName("progress")[0];

            var xhr = new XMLHttpRequest();
            upload = xhr.upload;

            // For speed measurements
            var lastLoaded;
            var lastTime;

            // Upload in progress
            upload.addEventListener("progress", function (e) {
                if (e.lengthComputable) {
                    progress.value = (e.loaded / e.total) * 100;
                    progress.max = 100;
                    progress.className = "progress progress-info progress-striped progress-animated";

                    var curTime = (new Date()).getTime();
                    if (e.loaded == e.total && e.total > 0) {
                        // Upload complete
                        speedText = "Server side processing... (" + filesize + ")";
                    } else if (lastTime !== 'undefined' && lastLoaded !== 'undefined') {
                        // Upload in progress
                        var bps = (e.loaded - lastLoaded) / (curTime - lastTime);
                        if (isNaN(bps)) {
                            speedText = "Uploading... (" + filesize + ")";
                        } else {
                            speedText = "Uploading at " + humanizeBytesPerSecond(bps) + " (" + filesize + ")";
                        }
                    } else {
                        // Upload just initiated
                        speedText = "(" + filesize + ")";
                    }

                    speed.textContent = speedText;
                    lastTime = curTime;
                    lastLoaded = e.loaded;
                }
            }, false);

            // Upload complete
            xhr.onload = function(e) {
                progress.value = 100;
                counter_uploading -= 1;
                if (xhr.status == 201 && xhr.readyState == 4) {
                    progress.className = "progress progress-success";
                    speed.textContent = "Complete (" + filesize + ")";
                    counter_completed += 1;
                } else {
                    progress.className = "progress progress-danger";
                    speed.textContent = "Failed with status " + xhr.status + " (" + filesize + ")";
                    console.log("Unexpected response code: " + xhr.status);
                    console.log("Response body: " + xhr.response);
                    counter_failed += 1;
                }
                updateFileCount();
            };

            // Handle upload errors here
            xhr.onerror = function (e) {
                console.log("onerror: status: " + xhr.status + ", readystate: " + xhr.readyState);
                console.log(e);
                progress.className = "progress progress-danger";
                progress.value = 100;
                speed.textContent = "Failed due to network error (" + filesize + ")";
                counter_failed += 1;
                counter_uploading -= 1;
                updateFileCount();
            };

            xhr.open(
                "POST",
                url
            );
            xhr.setRequestHeader("Cache-Control", "no-cache");
            xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
            xhr.setRequestHeader("Filename", file.name);
            xhr.setRequestHeader("Size", file.size);
            xhr.setRequestHeader("Tag", tag);
            xhr.send(file);
        }
    }
}

// http://stackoverflow.com/q/10420352
function getReadableFileSizeString(fileSizeInBytes) {
    var i = -1;
    var byteUnits = ['kB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    do {
        fileSizeInBytes = fileSizeInBytes / 1024;
        i++;
    } while (fileSizeInBytes > 1024);

    return Math.max(fileSizeInBytes, 0.1).toFixed(1) + byteUnits[i];
};

function deleteURL (url, messageBoxID) {
    console.log("Delete url: " + url);
    var xhr = new XMLHttpRequest();
    var box = document.getElementById(messageBoxID);

    xhr.onload = function(e) {
        if (xhr.status == 200 && xhr.readyState == 4) {
            console.log("Deleted successfully");
            box.textContent = "Delete operation completed successfully.";
            box.className = "alert alert-info";
        } else if (xhr.status  == 404 && xhr.readyState == 4) {
            box.textContent = "Not found.";
            box.className = "alert alert-info";
        } else {
            console.log("Failed to delete");
            box.textContent = "Error " + xhr.status + ". Unable to verify the operation.";
            box.className = "alert alert-danger";
        }
    };

    xhr.onerror = function (e) {
        console.log("onerror: status: " + xhr.status + ", readystate: " + xhr.readyState);
    };

    xhr.open(
        "DELETE",
        url
    );

    xhr.send();
};
