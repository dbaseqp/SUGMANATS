function createToast(message, color, delay) {
    const toastContainer = document.getElementById('toast-container');

    // Create a new toast element
    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.innerHTML = `
        <div class="toast-header ${color}">
        <strong class="me-auto">Notification</strong>
        <button type="button" class="btn-close" data-bs-dismiss="toast"></button>
        </div>
        <div class="toast-body">
        ${message}
        </div>
    `;

    // Add the toast to the container
    toastContainer.appendChild(toast);

    // Initialize the Bootstrap toast
    const bootstrapToast = new bootstrap.Toast(toast);

    // Show the toast
    bootstrapToast.show();

    // Automatically hide the toast after the specified delay
    if (delay) {
        setTimeout(() => {
        bootstrapToast.hide();
        }, delay);
    }
}

function postAjax(e, formid, data, url, success_function) {
    e.preventDefault();
    if (data instanceof FormData) {
        var form_data = data
    } else {
        var form_data = new FormData();
        for ( var key in data ) {
            form_data.append(key, data[key]);
        }
    }

    fetch(url, {
        method: "POST",
        body: form_data
    })
    .then(response => response.json())
    .then(response => {
        if (response.status == false) {
            createToast(response.message, "bg-danger")
        } else {
            success_function(response)
        }
    })
    .catch(error => {
        createToast(error, "bg-danger")
    })
}