document.addEventListener("DOMContentLoaded", function() {
fetch('/version')
    .then(response => response.text())  // Read the response as plain text
    .then(data => {
    document.querySelector('.cliversion').textContent = data;
    })
fetch('/contexts')
    .then(response => response.text())  // Read the response as plain text
    .then(data => {
        // Parse the data by splitting the text by lines
        const lines = data.trim().split('\n');

        // Filter out the line that indicates the start of the contexts
        const contexts = lines.slice(1).map(line => line.trim());

        // Get the container where buttons will be added
        const container = document.querySelector('.currentcontext');
        container.innerHTML = ''; // Clear any existing content

        // Loop through each context and create a button
        contexts.forEach(context => {
            const button = document.createElement('button');

            // Check if the context is the active one (starts with '*')
            if (context.startsWith('*')) {
                button.textContent = context.replace('*', '').trim(); // Remove the '*' from the text
                button.classList.add('active'); // Optional: Add a class to style active context
            } else {
                button.textContent = context;
            }

            // Add an event listener to the button to call changeContext() when clicked
            button.addEventListener('click', () => changeContext(button.textContent));

            // Append the button to the container
            container.appendChild(button);
        });
    })
    .catch(error => console.error('Error:', error));
});