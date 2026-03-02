<?php
  class User {
    private $db;

    public function __construct(){
      $this->db = new Database;
    }

    public function updateToken($user_id, $token, $tag)
    {
        $this->db->query('UPDATE  initkey_rid SET bearer_token = :bearer_token WHERE user_id= :user_id and email = :email');
        $this->db->bind(':user_id', $user_id);
        $this->db->bind(':bearer_token', $token);
        $this->db->bind(':email', $tag);
        // Execute
        if ($this->db->execute()) {
            return true;
        } else {
            return false;
        }
    }
      public function updatePassword($data)
    {
        $tokenTime = 0;
        $dateTime = new DateTime();
        $currentDate = $dateTime->format('Y-m-d H:i:s');

        $this->db->query('UPDATE  initkey_rid SET password_reset_token = :password_reset_token,password_reset_token_time = :password_reset_token_time, password = :password  WHERE email = :email AND user_id = :user_id ');
        $this->db->bind(':email', $data['email']);
        $this->db->bind(':user_id', $data['user_id']);
        $this->db->bind(':password_reset_token', 0);
        $this->db->bind(':password', $data['password']);
        $this->db->bind(':password_reset_token_time', $tokenTime);

        // Execute
        if ($this->db->execute()) {
            return true;
        } else {
            return false;
        }
    }
public function findUserByEmail1($email)
{
    $this->db->query("SELECT * FROM initkey_rid WHERE email = :email");

    // Bind Values
    $this->db->bind(':email', $email);

    $row = $this->db->single();

    // Check row
    if ($this->db->rowCount() > 0) {
        return true;
    } else {
        return false;
    }
}
    public function findAllUsers()
    {
        $this->db->query("SELECT * FROM initkey_rid");

        $rows = $this->db->resultSet();

        // Check roow
        return $rows;

    }
 public function updateResetToken($data)
    {
        $tokenTime = 15;
        $dateTime = new DateTime();
        $currentDate = $dateTime->format('Y-m-d H:i:s');

        $this->db->query('UPDATE  initkey_rid SET password_reset_token = :password_reset_token,password_reset_token_time = :password_reset_token_time, reset_token_set_date = :reset_token_set_date  WHERE email = :email ');
        $this->db->bind(':email', $data['email']);
        $this->db->bind(':password_reset_token', $data['otp']);
        $this->db->bind(':reset_token_set_date', $currentDate);
        $this->db->bind(':password_reset_token_time', $tokenTime);
        // Execute
        if ($this->db->execute()) {
            return true;
        } else {
            return false;
        }
    }
    public function findLoginByToken($token)
    {
        $this->db->query('SELECT * FROM initkey_rid WHERE bearer_token= :bearer_token');
        $this->db->bind(':bearer_token', $token);

        $row = $this->db->single();
    
        
        if($this->db->rowCount() > 0){
            return $row;
        } else{
            return false;
        
        
        
        }


    }

    
    public function loginUser($email)
    {
        $this->db->query('SELECT * FROM initkey_rid  WHERE email= :email');
        $this->db->bind(':email', $email);
        // $this->db->bind(':user_tag', $email);

        $row = $this->db->single();
       
        //return $row;
        if($this->db->rowCount() > 0){
            return $row;
        } else {
          
           return false;
       
        
        }


    }
    public function findUserByEmail_det($email)
    {
        $this->db->query("SELECT * FROM initkey_rid WHERE  email = :email");

        // Bind Values
        $this->db->bind(':email', $email);
        $row = $this->db->single();
        if($this->db->rowCount() > 0){
        return $row;
        }else{
            return false;
        }
    
    }
    // Add User / Register for Real Estate
    public function register($data){
      // Prepare Query

      $this->db->query('INSERT INTO initkey_rid (
        email,
        user_id,
        password,
        full_name,
        phone,
        user_type,
        company_name,
        license_number,
        bio,
        profile_image,
        address,
        city,
        state,
        country,
        created_at,
        activation
      ) VALUES (
        :email,
        :user_id,
        :password,
        :full_name,
        :phone,
        :user_type,
        :company_name,
        :license_number,
        :bio,
        :profile_image,
        :address,
        :city,
        :state,
        :country,
        NOW(),
        0
      )');

      // Bind Values
      $this->db->bind(':email', $data['email']);
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':password', $data['password']);
      $this->db->bind(':full_name', $data['full_name']);
      $this->db->bind(':phone', $data['phone']);
      $this->db->bind(':user_type', $data['user_type']);
      $this->db->bind(':company_name', $data['company_name']);
      $this->db->bind(':license_number', $data['license_number']);
      $this->db->bind(':bio', $data['bio']);
      $this->db->bind(':profile_image', $data['profile_image']);
      $this->db->bind(':address', $data['address']);
      $this->db->bind(':city', $data['city']);
      $this->db->bind(':state', $data['state']);
      $this->db->bind(':country', $data['country']);

      //Execute
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }

    }
    public function editUser($data){
      // Prepare Query for Real Estate Profile Update

      $this->db->query('UPDATE initkey_rid SET
        full_name = :full_name,
        email = :email,
        phone = :phone,
        user_type = :user_type,
        company_name = :company_name,
        license_number = :license_number,
        bio = :bio,
        profile_image = :profile_image,
        address = :address,
        city = :city,
        state = :state,
        country = :country,
        updated_at = NOW()
        WHERE user_id = :user_id');

      // Bind Values
      $this->db->bind(':full_name', $data['full_name']);
      $this->db->bind(':email', $data['email']);
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':phone', $data['phone']);
      $this->db->bind(':user_type', $data['user_type']);
      $this->db->bind(':company_name', $data['company_name']);
      $this->db->bind(':license_number', $data['license_number']);
      $this->db->bind(':bio', $data['bio']);
      $this->db->bind(':profile_image', $data['profile_image']);
      $this->db->bind(':address', $data['address']);
      $this->db->bind(':city', $data['city']);
      $this->db->bind(':state', $data['state']);
      $this->db->bind(':country', $data['country']);

      //Execute
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    }

    // Update User Profile
    public function updateProfile($data) {
      try {
        $this->db->query('UPDATE initkey_rid SET
                         full_name = :full_name,
                         phone = :phone,
                         company_name = :company_name,
                         license_number = :license_number,
                         bio = :bio,
                         profile_image = COALESCE(:profile_image, profile_image),
                         address = :address,
                         city = :city,
                         state = :state,
                         country = :country,
                         updated_at = NOW()
                         WHERE user_id = :user_id');

        $this->db->bind(':user_id', $data['user_id']);
        $this->db->bind(':full_name', $data['full_name']);
        $this->db->bind(':phone', $data['phone']);
        $this->db->bind(':company_name', $data['company_name']);
        $this->db->bind(':license_number', $data['license_number']);
        $this->db->bind(':bio', $data['bio']);
        $this->db->bind(':profile_image', $data['profile_image'] ?? null);
        $this->db->bind(':address', $data['address']);
        $this->db->bind(':city', $data['city']);
        $this->db->bind(':state', $data['state']);
        $this->db->bind(':country', $data['country']);

        if ($this->db->execute()) {
          return true;
        } else {
          return false;
        }
      } catch (Exception $e) {
        error_log('Update profile error: ' . $e->getMessage());
        return false;
      }
    }

    // Delete User Profile
    public function deleteUser($user_id){
      try {
        // First delete related data (listings, wishlist, etc.)
        $this->db->query('DELETE FROM listings WHERE user_id = :user_id');
        $this->db->bind(':user_id', $user_id);
        $this->db->execute();

        $this->db->query('DELETE FROM wishlist WHERE user_id = :user_id');
        $this->db->bind(':user_id', $user_id);
        $this->db->execute();

        // Delete user account
        $this->db->query('DELETE FROM initkey_rid WHERE user_id = :user_id');
        $this->db->bind(':user_id', $user_id);

        if($this->db->execute()){
          return true;
        } else {
          return false;
        }
      } catch (PDOException $e) {
        return false;
      }
    }

    // Update user password
    public function updateUserPassword($user_id, $new_password){
      $this->db->query('UPDATE initkey_rid SET password = :password, updated_at = NOW() WHERE user_id = :user_id');

      $this->db->bind(':password', $new_password);
      $this->db->bind(':user_id', $user_id);

      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    }

    // Get user profile by ID
    public function getUserProfile($user_id){
      $this->db->query("SELECT user_id, email, full_name, phone, user_type, company_name, license_number, bio, profile_image, address, city, state, country, created_at, updated_at FROM initkey_rid WHERE user_id = :user_id");
      $this->db->bind(':user_id', $user_id);

      $row = $this->db->single();

      return $row;
    }

    // Check if user is agent/realtor
    public function isAgent($user_id){
      $this->db->query("SELECT user_type FROM initkey_rid WHERE user_id = :user_id");
      $this->db->bind(':user_id', $user_id);

      $row = $this->db->single();

      return $row && ($row->user_type === 'agent' || $row->user_type === 'realtor');
    }

    // Check if user is verified
    public function isVerified($user_id){
      $this->db->query("SELECT activation FROM initkey_rid WHERE user_id = :user_id");
      $this->db->bind(':user_id', $user_id);

      $row = $this->db->single();

      return $row && $row->activation == 1;
    }

    // Verify user account (admin function)
    public function verifyUser($user_id){
      try {
        $this->db->query('UPDATE initkey_rid SET activation = 1, updated_at = NOW() WHERE user_id = :user_id');
        $this->db->bind(':user_id', $user_id);

        if($this->db->execute()){
          return true;
        } else {
          return false;
        }
      } catch (PDOException $e) {
        return false;
      }
    }

    // Find user by ID
    public function findUserById($user_id){
      try {
        $this->db->query('SELECT * FROM initkey_rid WHERE user_id = :user_id');
        $this->db->bind(':user_id', $user_id);

        $row = $this->db->single();

        if($row){
          return $row;
        } else {
          return false;
        }
      } catch (PDOException $e) {
        return false;
      }
    }

    // Generate password reset token
    public function generatePasswordResetToken($email){
      try {
        $token = sprintf('%06d', mt_rand(100000, 999999)); // 6-digit token
        $expiry = time() + (30 * 60); // 30 minutes from now

        $this->db->query('UPDATE initkey_rid SET
                         password_reset_token = :token,
                         password_reset_token_time = :expiry,
                         reset_token_set_date = NOW()
                         WHERE email = :email');

        $this->db->bind(':token', password_hash($token, PASSWORD_DEFAULT));
        $this->db->bind(':expiry', $expiry);
        $this->db->bind(':email', $email);

        if($this->db->execute()){
          return $token; // Return plain token for email
        } else {
          return false;
        }
      } catch (PDOException $e) {
        return false;
      }
    }

    // Reset password with token
    public function resetPasswordWithToken($email, $token, $new_password){
      try {
        // Get user's stored token and expiry
        $this->db->query('SELECT password_reset_token, password_reset_token_time FROM initkey_rid WHERE email = :email');
        $this->db->bind(':email', $email);
        $user = $this->db->single();

        if(!$user || !$user->password_reset_token){
          return false; // No reset token found
        }

        // Check if token is expired
        if(time() > $user->password_reset_token_time){
          return false; // Token expired
        }

        // Verify token
        if(!password_verify($token, $user->password_reset_token)){
          return false; // Invalid token
        }

        // Update password and clear reset token
        $this->db->query('UPDATE initkey_rid SET
                         password = :password,
                         password_reset_token = NULL,
                         password_reset_token_time = 0,
                         reset_token_set_date = NULL,
                         updated_at = NOW()
                         WHERE email = :email');

        $this->db->bind(':password', $new_password);
        $this->db->bind(':email', $email);

        return $this->db->execute();
      } catch (PDOException $e) {
        return false;
      }
    }
    // Verify password reset token
    public function verifyPasswordResetToken($email, $token) {
      try {
        $this->db->query("SELECT user_id, password_reset_token, password_reset_token_time
                         FROM initkey_rid
                         WHERE email = :email AND password_reset_token = :token");

        $this->db->bind(':email', $email);
        $this->db->bind(':token', $token);

        $row = $this->db->single();

        if ($row && $row->password_reset_token_time > time()) {
          return $row;
        } else {
          return false;
        }
      } catch (PDOException $e) {
        return false;
      }
    }

    

    // Upgrade user type (buyer to agent/realtor)
    public function upgradeUserType($user_id, $newUserType, $additionalData = []) {
      try {
        $validTypes = ['buyer', 'agent', 'realtor'];
        if (!in_array($newUserType, $validTypes)) {
          return false;
        }

        $this->db->query('UPDATE initkey_rid SET
          user_type = :user_type,
          company_name = :company_name,
          license_number = :license_number,
          updated_at = NOW()
          WHERE user_id = :user_id');

        $this->db->bind(':user_type', $newUserType);
        $this->db->bind(':company_name', $additionalData['company_name'] ?? '');
        $this->db->bind(':license_number', $additionalData['license_number'] ?? '');
        $this->db->bind(':user_id', $user_id);

        if($this->db->execute()){
          return true;
        } else {
          return false;
        }
      } catch (PDOException $e) {
        return false;
      }
    }
    public function saveaccount($data){
      // Prepare Query
      
      $this->db->query('INSERT INTO  virtual_accounts (user_id, account_name,account_ref,email,account_number) 
      VALUES (:user_id, :account_name, :account_ref, :email, :account_number)');
      // Bind Values
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':account_name', $data['accountName']);
      $this->db->bind(':account_ref', $data['accountReference']);
      $this->db->bind(':email', $data['customerEmail']);
      $this->db->bind(':account_number', $data['account_number']);
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    
    }

    // Find USer BY Email
    public function findUserByEmail($email){
       $this->db->query("SELECT * FROM initkey_rid WHERE email = :email");
      $this->db->bind(':email', $email);

      $row = $this->db->single();

      //Check Rows
      if($this->db->rowCount() > 0){
        return true;
      } else {
        return false;
      }
    }
    public function findUserByEmail2($email){
       $this->db->query("SELECT * FROM initkey WHERE email = :email");
      $this->db->bind(':email', $email);

      $row = $this->db->single();

      //Check Rows
      if($this->db->rowCount() > 0){
        return $row;
      } else {
        return false;
      }
    }
    public function getVirtualAccount($email){
       $this->db->query("SELECT * FROM virtual_accounts WHERE user_id = :user_id");
      $this->db->bind(':user_id', $email);

      $row = $this->db->single();

      //Check Rows
      if($this->db->rowCount() > 0){
        return $row;
      } else {
        return false;
      }
    }
    public function findUserByOtp($email){
       $this->db->query("SELECT * FROM initkey WHERE otp = :otp");
      $this->db->bind(':otp', $email);

      $row = $this->db->single();

      //Check Rows
      if($this->db->rowCount() > 0){
        return true;
      } else {
        return false;
      }
    }
    public function updateOTP($email, $otp)
{
    // Prepare SQL statement
    $this->db->query('UPDATE initkey SET otp = :otp WHERE email = :email');

    // Bind values
    $this->db->bind(':email', $email);
    $this->db->bind(':otp', $otp);

    // Execute the query
    if ($this->db->execute()) {
        return true;
    } else {
        return false;
    }
}
public function updateOTP2($oldOTP)
{
    // Prepare SQL statement
    $sql = 'UPDATE initkey SET otp = :newOTP WHERE otp = :oldOTP';

    // Bind values
    $this->db->query($sql);
    $this->db->bind(':newOTP', '0');
    $this->db->bind(':oldOTP', $oldOTP);
    // Execute the query
    if ($this->db->execute()) {
        return true;
    } else {
        return false;
    }
}

public function login2($email, $pass)
{
    // Prepare SQL statement
    $sql = 'UPDATE initkey SET password = :password WHERE email = :email';

    // Bind values
    $this->db->query($sql);
    $this->db->bind(':password', $pass);
    $this->db->bind(':email', $email);
    // Execute the query
    if ($this->db->execute()) {
        return true;
    } else {
        return false;
    }
}


    // Login / Authenticate User
    public function login($email, $password){
      $this->db->query("SELECT * FROM initkey WHERE email = :email");
      $this->db->bind(':email', $email);

      $row = $this->db->single();
      
      $hashed_password = $row->password;
      if(password_verify($password, $hashed_password)){
        return $row;
      } else {
        return false;
      }
    }
    public function getbalance($uid){
      $this->db->query("SELECT * FROM user_account WHERE user_id = :user_id");
      $this->db->bind(':user_id', $uid);

      $row = $this->db->single();

      if($row){
        return $row;
      } else {
        return false;
      }
    }

    // Find User By ID
    public function getUserById($id){
      $this->db->query("SELECT * FROM initkey_rid WHERE user_id = :user_id");
      $this->db->bind(':user_id', $id);

      $row = $this->db->single();

      return $row;
    }
    public function getUser($id){
      $this->db->query("SELECT * FROM initkey_rid WHERE user_id = :user_id");
      $this->db->bind(':user_id', $id);

      $row = $this->db->single();

      return $row;
    }
  }