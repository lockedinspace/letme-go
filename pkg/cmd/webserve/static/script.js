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
            const itemDiv = document.createElement('button');
            itemDiv.classList.add('button-14'); 
            itemDiv.textContent = item.name; 

            itemDiv.onclick = () => obtainCredentials(item.name);

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
                        body: JSON.stringify({ context: accountName, mfaToken: Number(mfaToken), credentialProcess: credentialProcess(), renew: renew() }),
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
    const accountItems = document.getElementsByClassName('button-14');
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
function credentialProcess(){
    return document.getElementById('credentialprocess').checked;
}
function renew(){
    return document.getElementById('renew').checked;
}