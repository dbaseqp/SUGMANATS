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