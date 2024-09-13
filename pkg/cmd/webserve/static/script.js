document.addEventListener("DOMContentLoaded", function() { 
// fetch('/version')
//     .then(response => response.text())  // Read the response as plain text
//     .then(data => {
//     document.querySelector('.version').textContent = data;
//     })
fetch('/list')
    .then(response => response.json())  // Parse the response as JSON
    .then(data => {
        const container = document.querySelector('.accounts-list');

        data.items.forEach(item => {
            // Create a div to hold the item name and buttons
            const itemDiv = document.createElement('div');
            itemDiv.classList.add('item-container');  // Add a class for styling if needed

            // Create a span to hold the item name
            const itemName = document.createElement('span');
            itemName.textContent = item.name;  // Set the text to the account name
            itemName.classList.add('account-name');  // Optional class for styling

            // Create the CLI button
            const cliButton = document.createElement('button');
            cliButton.classList.add('button-14'); 
            cliButton.textContent = 'CLI';
            cliButton.onclick = () => obtainCredentials(item.name);  // Set onclick for CLI button

            // Create the Federated button
            const federatedButton = document.createElement('button');
            federatedButton.classList.add('button-14'); 
            federatedButton.textContent = 'Federated';
            federatedButton.onclick = () => obtainFederatedCredentials(item.name);  // Set onclick for Federated button

            // Append the item name and both buttons to the div
            itemDiv.appendChild(itemName);
            itemDiv.appendChild(cliButton);
            itemDiv.appendChild(federatedButton);

            // Append the item div to the main container
            container.appendChild(itemDiv);
        });
    })
    .catch(error => console.error('Error:', error));
fetch('/active-accounts')
    .then(response => response.json())  // Parse the response as JSON
    .then(data => {
        const container = document.querySelector('.active-accounts');
        const currentTime = Math.floor(Date.now() / 1000); // Current epoch time in seconds

        // Iterate over each item in the data
        for (const [key, value] of Object.entries(data)) {
            const expiryTime = value.expiry;
            const timeLeft = expiryTime - currentTime;

            // Calculate time left in minutes and seconds
            const minutesLeft = Math.floor(timeLeft / 60);
            const secondsLeft = timeLeft % 60;

            // Format expiry time as a human-readable string
            const expiryDate = new Date(expiryTime * 1000);
            const expiryTimeString = expiryDate.toLocaleTimeString();

            // Create a readable expiry message
            const message = `${key} - expires in ${minutesLeft} mins at ${expiryTimeString}`;

            // Create a div element to display the message
            const expiryItem = document.createElement('div');
            expiryItem.classList.add('expiry-item');

            // Add label and time info
            const label = document.createElement('div');
            label.classList.add('label');
            label.textContent = message;
            expiryItem.appendChild(label);

            container.appendChild(expiryItem);
        }
    })
    .catch(error => console.error('Error:', error));
fetch('/context-values')
    .then(response => response.text())  // Read the response as plain text
    .then(data => {
        const container = document.querySelector('.active-context-values');

        // Create a new div for the fetched data
        const dataDiv = document.createElement('div');
        dataDiv.textContent = data;

        // Append the new data below the existing <h3>
        container.appendChild(dataDiv); // Append the new data
    })
    .catch(error => console.error('Error:', error));
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
            button.classList.add('button-13'); 

            // If the context is active, set the button background color to green
            if (active === "true") {
                button.classList.add('button-15'); // You can adjust the shade of green as desired
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
function obtainFederatedCredentials(accountName) {
    console.log(`Obtaining federated credentials for: ${accountName}`);

    fetch('/context-values')
        .then(response => response.json())
        .then(data => {
            const mfaArn = data.AwsMfaArn;
            if (mfaArn && mfaArn.length > 0) {
                const mfaToken = prompt("Please enter your MFA token:");
                if (mfaToken) {
                    const requestBody = { context: accountName, mfaToken: mfaToken };
                    console.log('Request Body:', JSON.stringify(requestBody));

                    fetch('/obtain-federated', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify(requestBody),
                    })
                    .then(response => {
                        if (!response.ok) {
                            throw new Error(`HTTP error! status: ${response.status}`);
                        }
                        const contentType = response.headers.get('Content-Type');
                        if (contentType && contentType.includes('application/json')) {
                            return response.json(); // Parse JSON only if valid
                        } else {
                            return response.text(); // Handle non-JSON response (like error message)
                        }
                    })
                    .then(data => {
                        if (typeof data === 'string') {
                            console.error('Non-JSON response:', data);
                            alert(`Error: ${data}`);
                        } else {
                            console.log('Raw Response:', data);
                            if (data.aws_console_sign_in_url) {
                                displayModal(data.aws_console_sign_in_url);
                            } else {
                                console.error('Sign-in URL not found in the response.');
                            }
                        }
                    })
                    .catch(error => console.error('Error:', error));
                } else {
                    console.error('MFA token is required.');
                }
            } else {
                const requestBody = { context: accountName };
                console.log('Request Body:', JSON.stringify(requestBody));

                fetch('/obtain-federated', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(requestBody),
                })
                .then(response => {
                    if (!response.ok) {
                        throw new Error(`HTTP error! status: ${response.status}`);
                    }
                    const contentType = response.headers.get('Content-Type');
                    if (contentType && contentType.includes('application/json')) {
                        return response.json(); // Parse JSON only if valid
                    } else {
                        return response.text(); // Handle non-JSON response (like error message)
                    }
                })
                .then(data => {
                    if (typeof data === 'string') {
                        console.error('Non-JSON response:', data);
                        alert(`Error: ${data}`);
                    } else {
                        console.log('Raw Response:', data);
                        if (data.aws_console_sign_in_url) {
                            displayModal(data.aws_console_sign_in_url);
                        } else {
                            console.error('Sign-in URL not found in the response.');
                        }
                    }
                })
                .catch(error => console.error('Error:', error));
            }
        })
        .catch(error => console.error('Error fetching context values:', error));
}
function obtainCredentials(accountName) {
    console.log(`Obtaining credentials for: ${accountName}`);
    // Check if len(mfa_arn) > 0, prompt for MFA token, else go without MFA token
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
                        // Keep mfaToken as a string to avoid stripping leading zeros
                        body: JSON.stringify({ context: accountName, mfaToken: mfaToken, credentialProcess: credentialProcess(), renew: renew() }),
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
                    body: JSON.stringify({ context: accountName, credentialProcess: credentialProcess(), renew: renew() }),
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
    const accountItems = document.getElementsByClassName('item-container');
    
    for (let item of accountItems) {
        // Get the text content of the span with class 'account-name' within each item
        const accountName = item.querySelector('.account-name').textContent.toLowerCase();

        // Create a regex pattern for fuzzy matching
        const pattern = new RegExp(input.split('').join('.*'), 'i');

        // Check if the account name matches the pattern
        if (pattern.test(accountName)) {
            item.style.display = ''; // Show the item if it matches
        } else {
            item.style.display = 'none'; // Hide the item if it doesn't match
        }
    }
}
function credentialProcess(){
    return document.getElementById('credentialprocess').checked;
}
function renew(){
    return document.getElementById('renew').checked;
}
function displayModal(url) {
    document.getElementById('signInUrl').innerText = url;
    document.getElementById('urlModal').style.display = 'block'; // Show the modal
}

function closeModal() {
    document.getElementById('urlModal').style.display = 'none'; // Hide the modal
}

function openInNewTab() {
    const url = document.getElementById('signInUrl').innerText;
    window.open(url, '_blank');
}

function copyToClipboard() {
    const urlText = document.getElementById('signInUrl').innerText;
    navigator.clipboard.writeText(urlText).then(() => {
        alert('URL copied to clipboard!');
    }).catch(err => {
        console.error('Failed to copy the text: ', err);
    });
}