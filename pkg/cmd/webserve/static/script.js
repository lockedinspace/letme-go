document.addEventListener("DOMContentLoaded", function() { 
fetch('/version')
    .then(response => response.text())  // Read the response as plain text
    .then(data => {
    document.querySelector('.version').textContent = data;
    })
fetch('/list')
    .then(response => response.json())  // Parse the response as JSON
    .then(data => {
        const container = document.querySelector('.accounts-list');

        // Iterate over the items and create a div for each one
        data.items.forEach(item => {
            const itemDiv = document.createElement('div');
            itemDiv.classList.add('account-item'); // Add a class for styling
            itemDiv.textContent = item.name; // Set the name as the text content

            // Set up the click event to call obtainCredentials
            itemDiv.onclick = () => obtainCredentials(item.name);

            // Append the div to the container
            container.appendChild(itemDiv);
        });
    })
    .catch(error => console.error('Error:', error));
fetch('/context-values')
    .then(response => response.text())  // Read the response as plain text
    .then(data => {
        document.querySelector('.active-context-values').textContent = data;
    })
fetch('/contexts')
    .then(response => response.json())  // Parse the response as JSON
    .then(data => {
        // Get the container where buttons will be added
        const container = document.querySelector('.contexts');

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
    fetch(`/switch-context`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ context: contextName }),
    })
    .then(response => {
        if (response.ok) {
            // If the switch was successful, reload the page
            location.reload();
        } else {
            console.error('Failed to switch context:', response.statusText);
        }
    })
    .catch(error => console.error('Error:', error));
}
function obtainCredentials(accountName) {
    console.log(`Obtaining credentials for: ${accountName}`);
    //check if len(mfa_arn) > 0, prompt for mfa token, else go without mfa token
    fetch(`/context-values`)
        .then(response => response.json())
        .then(data => {
            const mfaArn = data.AwsMfaArn;
            // Check if AwsMfaArn is empty or not
            if (mfaArn && mfaArn.length > 0) {
                // Prompt the user for MFA token input
                const mfaToken = prompt("Please enter your MFA token:");
                
                // If a token was provided, include it in the request
                if (mfaToken) {
                    fetch(`/obtain`, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({ context: accountName, mfaToken: Number(mfaToken) }),
                    })
                    .then(response => {
                        if (response.ok) {
                            // If the switch was successful, reload the page
                            location.reload();
                        } else {
                            console.error('Failed to switch context:', response.statusText);
                        }
                    })
                    .catch(error => console.error('Error:', error));
                } else {
                    console.error('MFA token is required.');
                }
            } else {
                fetch(`/obtain`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ context: accountName }),
                })
                .then(response => {
                    if (response.ok) {
                        // If the switch was successful, reload the page
                        location.reload();
                    } else {
                        console.error('Failed to switch context:', response.statusText);
                    }
                })
                .catch(error => console.error('Error:', error));
            }
        })
        .catch(error => console.error('Error fetching context values:', error));
}

function filterAccounts() {
    const input = document.getElementById('searchBar').value.toLowerCase().trim();
    const accountItems = document.getElementsByClassName('account-item');

    for (let item of accountItems) {
        // Create a regex pattern for fuzzy matching
        const pattern = new RegExp(input.split('').join('.*'), 'i');

        // Check if the item text matches the pattern
        if (pattern.test(item.textContent.toLowerCase())) {
            item.style.display = ''; // Show the item if it matches
        } else {
            item.style.display = 'none'; // Hide the item if it doesn't match
        }
    }
}