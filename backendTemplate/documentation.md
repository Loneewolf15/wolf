Marketplaza API Endpoints Documentation

Here is the documentation for the endpoints we've created for the Marketplaza users API, including the expected fields and data format for each request.

### **baseUrl = https://api.marketplaza.app
### **requestID = pid2025.... must follow every POST request to the API

### **1. User Registration**

* **Endpoint:** `POST /users/registerUser`
* **Description:** Creates a new user account with email and password. This is the first step in the traditional user sign-up process.
* **Request Body (JSON):**
    * `name`: `string`, **required**. The user's full name.
    * `email`: `string`, **required**. A valid and unique email address.
    * `phone`: `string`, **required**. A unique phone number.
    * `password`: `string`, **required**. The user's password (min. 6 characters).
    * `confirm_password`: `string`, **required**. Must match the password field.

***

### **2. User Login**

* **Endpoint:** `POST /users/loginfunc`
* **Description:** Authenticates a user and provides a JWT for session management.
* **Request Body (JSON):**
    * `email`: `string`, **required**. The user's registered email.
    * `password`: `string`, **required**. The user's password.

***

### **3. Social Login & Sign-up**

* **Endpoint:** `POST /users/socialLogin`
* **Description:** Handles both login and registration for users authenticating via social providers (Google, Apple, Facebook).
* **Request Body (JSON):**
    * `provider`: `string`, **required**. The social media platform (`google`, `facebook`, or `apple`).
    * `social_id`: `string`, **required**. The unique ID provided by the social platform.
    * `token`: `string`, **required**. The ID token or access token from the social provider.
    * `email`: `string`, **required**. The user's email from the social provider.
    * `name`: `string`, **required**. The user's full name from the social provider.

***

### **4. Get User Profile**

* **Endpoint:** `GET /users/getUser`
* **Description:** Retrieves the profile information of the authenticated user.
* **Request Headers:**
    * `Authorization`: `Bearer <JWT>`. The JWT obtained from a successful login or registration.
* **Request Body:** None.

***

### **5. Update User Profile**

* **Endpoint:** `POST /users/editProfile`
* **Description:** Allows an authenticated user to update their profile information.
* **Request Headers:**
    * `Authorization`: `Bearer <JWT>`.
* **Request Body (form-data or JSON):**
    * `name`: `string`, **required**. The user's full name.
    * `phone`: `string`, **required**. The user's phone number.
    * `location`: `string`, **required**. The user's address.
    * `profile_image`: `file`, **optional**. An image file (JPEG, PNG, GIF, WebP) to upload as the profile picture.

***

### **6. Update User Location**

* **Endpoint:** `POST /users/updateLocation`
* **Description:** Stores a user's geographical location after they grant permission or manually enter their address.
* **Request Headers:**
    * `Authorization`: `Bearer <JWT>`.
* **Request Body (JSON):**
    * `address`: `string`, **required**. The user's full address.
    * `city`: `string`, **required**. The user's city.
    * `state`: `string`, **required**. The user's state.

***

### **7. Change Password**

* **Endpoint:** `POST /users/changePassword`
* **Description:** Allows an authenticated user to change their password.
* **Request Headers:**
    * `Authorization`: `Bearer <JWT>`.
* **Request Body (JSON):**
    * `current_password`: `string`, **required**. The user's current password for verification.
    * `new_password`: `string`, **required**. The new password (min. 6 characters).
    * `confirm_password`: `string`, **required**. Must match the new password.

***

### **8. Forgot Password**

* **Endpoint:** `POST /users/forgotPassword`
* **Description:** Sends a password reset token to the user's email.
* **Request Body (JSON):**
    * `email`: `string`, **required**. The email address associated with the account.

***

### **9. Reset Password**

* **Endpoint:** `POST /users/resetPassword`
* **Description:** Resets the user's password using the token sent to their email.
* **Request Body (JSON):**
    * `email`: `string`, **required**. The user's email address.
    * `reset_token`: `string`, **required**. The token received in the password reset email.
    * `new_password`: `string`, **required**. The new password.
    * `confirm_password`: `string`, **required**. Must match the new password.

***

### **10. Upgrade Account**

* **Endpoint:** `POST /users/upgradeAccount`
* **Description:** Upgrades a user from a "Buyer" account to another role, such as a seller, reseller, or delivery agent.
* **Request Headers:**
    * `Authorization`: `Bearer <JWT>`.
* **Request Body (JSON):**
    * `user_type`: `string`, **required**. The new role (`seller`, `reseller`, `delivery_agent`).
    * `company_name`: `string`, **optional**. [cite_start]Required for certain roles like "realtor" in the original plan[cite: 91].
    * `license_number`: `string`, **optional**. [cite_start]Required for the "realtor" role[cite: 91].

***

### **11. Delete Account**

* **Endpoint:** `POST /users/deleteProfile`
* **Description:** Permanently deletes the user's account and all associated data.
* **Request Headers:**
    * `Authorization`: `Bearer <JWT>`.
* **Request Body (JSON):**
    * `password`: `string`, **required**. The user's current password for confirmation.
    * `confirmation`: `string`, **required**. Must be the literal string "DELETE" to confirm.

***

### **12. Logout**

* **Endpoint:** `POST /users/logout`
* **Description:** Invalidates the user's current JWT, effectively logging them out.
* **Request Headers:**
    * `Authorization`: `Bearer <JWT>`.
* **Request Body:** None.