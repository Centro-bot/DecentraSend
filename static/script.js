document.addEventListener("DOMContentLoaded", function () {
    const form = document.getElementById("uploadForm");
    const status = document.getElementById("status");
    const fileList = document.getElementById("fileList");
    const fileInput = document.getElementById("file");
    const fileInfo = document.getElementById("file-info");
    const progressContainer = document.getElementById("progress-container");
    const progressBar = document.getElementById("progress-bar");

    // Update file info
    fileInput.addEventListener("change", function () {
        const file = fileInput.files[0];
        if (file) {
            if (file.type === "application/pdf") {
                fileInfo.textContent = `File Selected: ${file.name}`;
            } else {
                fileInfo.textContent = "Please select a valid PDF file.";
                fileInfo.style.color = "red";
            }
        }
    });

    // Update file list display
    function updateFileList(filename) {
        const fileItem = document.createElement("div");
        fileItem.classList.add("file-item");
        fileItem.textContent = filename;
        fileList.appendChild(fileItem);
    }

    // Handle form submission
    form.addEventListener("submit", function (e) {
        e.preventDefault();

        const file = fileInput.files[0];

        if (!file) {
            status.textContent = "Please select a file.";
            status.style.color = "red";
            return;
        }

        if (file.type !== "application/pdf") {
            status.textContent = "Please upload a PDF file.";
            status.style.color = "red";
            return;
        }

        const formData = new FormData();
        formData.append("file", file);

        // Show progress bar
        progressContainer.style.display = "block";
        progressBar.style.width = "0%";
        status.textContent = "Uploading...";

        // Upload the file using fetch with progress tracking
        const xhr = new XMLHttpRequest();
        xhr.open("POST", "/upload", true);

        // Update progress bar
        xhr.upload.onprogress = function (e) {
            if (e.lengthComputable) {
                const percent = (e.loaded / e.total) * 100;
                progressBar.style.width = percent + "%";
            }
        };

        // Handle successful upload
        xhr.onload = function () {
            if (xhr.status === 200) {
                status.textContent = "Upload successful!";
                status.style.color = "green";
                updateFileList(file.name);
            } else {
                status.textContent = "Error uploading file.";
                status.style.color = "red";
            }
            progressContainer.style.display = "none"; // Hide progress bar
        };

        // Handle error
        xhr.onerror = function () {
            status.textContent = "Error uploading file.";
            status.style.color = "red";
            progressContainer.style.display = "none";
        };

        xhr.send(formData);
    });
});
