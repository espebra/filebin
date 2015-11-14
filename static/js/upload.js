function FileAPI (c, t, d, f, tag) {

    var fileCount = c,
        fileList = t,
        dropZone = d,
        fileField = f,
        counter_queue = 0,
        counter_uploading = 0,
        counter_completed = 0,
        counter_failed = 0,
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
        var text = counter_completed + " of " + counter_queue + " file";
        if (counter_queue != 1){
            text = text + "s";
        }
        text = text + " uploaded";
        if (counter_failed > 0) {
            fileCount.textContent = text + ". " + counter_failed + " failed";
        }
        if (counter_completed == counter_queue) {
            fileCount.textContent = text + ", all done!";
        } else {
              fileCount.textContent = text + "...";
        }
	document.getElementById('fileCount').style.display = 'block';
        
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
        while (fileQueue.length > 0) {
            var item = fileQueue.pop();
            uploadFile(item.file, item.container);
        }
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
            var container = document.createElement("li");
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
            var size = document.createElement("div");
            var sizeText = document.createTextNode(filesize);
            size.appendChild(sizeText);
	    size.className = "col-md-2";
            meta.appendChild(size)

            var mimetype = file.type;
            if (mimetype.length == 0){
                mimetype = "unknown";
            }
            var mime = document.createElement("div");
            var mimeText = document.createTextNode(mimetype);
            mime.appendChild(mimeText);
            mime.className = "col-md-2";
            meta.appendChild(mime)

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
            counter_uploading += 1;
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

    var uploadFile = function (file, container) {
        if (container && file) {
            var bar = container.getElementsByTagName("div")[5];
            console.log(bar);
            var progress = bar.getElementsByTagName("progress")[0];
            console.log(progress);

            var xhr = new XMLHttpRequest();
            upload = xhr.upload;

            // Upload in progress
            upload.addEventListener("progress", function (e) {
                if (e.lengthComputable) {
                    progress.value = (e.loaded / e.total) * 100;
                    progress.max = 100;
                    progress.className = "progress progress-info";
                    //tr.className = "table-info";
                }
            }, false);

            // Upload complete
            xhr.onload = function(e) {
                progress.value = 100;
                counter_uploading -= 1;
                if (xhr.status == 201 && xhr.readyState == 4) {
                    progress.className = "progress progress-success";
                    counter_completed += 1;
                } else {
                    progress.className = "progress progress-danger";
                    console.log("Unexpected response code: " + this.status);
                    console.log("Response body: " + this.response);
                    counter_failed += 1;
                }
                updateFileCount();
            };

            // Handle upload errors here
            xhr.onerror = function (e) {
                //bar.className = "progress progress-warning";
                //tr.className = "table-warning";
                console.log(e);
            };

            xhr.open(
                "POST",
                "/"
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
