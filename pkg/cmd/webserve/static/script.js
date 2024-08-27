document.addEventListener("DOMContentLoaded", function() { 
fetch('/version')
    .then(response => response.text())  // Read the response as plain text
    .then(data => {
    document.querySelector('.cliversion').textContent = data;
    })
fetch('/list')
    .then(response => response.json())  // Parse the response as JSON
    .then(data => {
        const container = document.querySelector('.accounts');
        container.innerHTML = ''; // Clear any existing content

        const totalItems = data.items.length;
        
        // Calculate the number of columns (e.g., 4 columns)
        const maxColumns = 4;
        const columnCount = Math.min(maxColumns, totalItems);
        const rowCount = Math.ceil(totalItems / columnCount);

        // Create the table element
        const table = document.createElement('table');
        table.classList.add('accounts');

        for (let row = 0; row < rowCount; row++) {
            const tableRow = document.createElement('tr');

            for (let col = 0; col < columnCount; col++) {
                const index = row + col * rowCount;
                const cell = document.createElement('td');

                if (index < totalItems) {
                    const button = document.createElement('button');
                    button.textContent = data.items[index].name;
                    button.onclick = () => obtainCredentials(data.items[index].name);
                    cell.appendChild(button);
                }
                
                tableRow.appendChild(cell);
            }

            table.appendChild(tableRow);
        }

        container.appendChild(table);
    })
    .catch(error => console.error('Error:', error));
fetch('/context-values')
    .then(response => response.text())  // Read the response as plain text
    .then(data => {
        document.querySelector('.currentcontextvalues').textContent = data;
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
    const tables = document.getElementsByClassName('accounts');
    
    // Assuming there's only one table with class 'accounts'
    const table = tables[0];
    const rows = table.getElementsByTagName('tr');

    for (let row of rows) {
        const cells = row.getElementsByTagName('td');
        let rowVisible = false;

        for (let cell of cells) {
            const button = cell.getElementsByTagName('button')[0];
            if (button) {
                // Create a regex pattern for fuzzy matching
                const pattern = new RegExp(input.split('').join('.*'), 'i');
                if (pattern.test(button.textContent)) {
                    cell.style.display = '';  // Show the cell
                    rowVisible = true;
                } else {
                    cell.style.display = 'none';  // Hide the cell
                }
            }
        }

        // Hide the row if no cells are visible
        row.style.display = rowVisible ? '' : 'none';
    }
}