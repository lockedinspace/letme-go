document.addEventListener("DOMContentLoaded", function() {
fetch('/version')
    .then(response => response.text())  // Read the response as plain text
    .then(data => {
    document.querySelector('.cliversion').textContent = data;
    })
fetch('/contexts')
    .then(response => response.json())  // Parse the response as JSON
    .then(data => {
        // Get the container where buttons will be added
        const container = document.querySelector('.currentcontext');
        container.innerHTML = ''; // Clear any existing content

        // Loop through each context item
        data.items.forEach(item => {
            const { name, active } = item;

            // Create a div to wrap the button
            const div = document.createElement('div');

            // Create the button for the context
            const button = document.createElement('button');
            button.textContent = name;

            // If the context is active, set the button background color to green
            if (active === "true") {
                button.style.backgroundColor = 'lightgreen'; // You can adjust the shade of green as desired
            }

            // Add an event listener to the button to call changeContext() when clicked
            button.addEventListener('click', () => changeContext(name));

            // Append the button to the div, and then append the div to the container
            div.appendChild(button);
            container.appendChild(div);
        });
    })
    .catch(error => console.error('Error:', error));
});

function changeContext(contextName) {
    console.log(`Changing context to: ${contextName}`);
}
