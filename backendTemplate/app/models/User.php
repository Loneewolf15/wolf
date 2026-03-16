<?php
class User {
  private $db;

  public function __construct(){
    $this->db = new Database;
  }

  // Find a user by their email and return all details
  public function findUserByEmailAndGetDetails($email) {
    try {
      $this->db->query("SELECT * FROM initkey_rid WHERE email = :email");
      $this->db->bind(':email', $email);
      $row = $this->db->single();
      if ($this->db->rowCount() > 0) {
        return $row;
      } else {
        return false;
      }
    } catch (Exception $e) {
      error_log('findUserByEmailAndGetDetails error: ' . $e->getMessage());
      return false;
    }
  }

  public function findUserByEmailOrPhone($username) {
    try {
      $this->db->query("SELECT * FROM initkey_rid WHERE email = :username OR phone = :username");
      $this->db->bind(':username', $username);
      $row = $this->db->single();
      if ($this->db->rowCount() > 0) {
        return $row;
      } else {
        return false;
      }
    } catch (Exception $e) {
      error_log('findUserByEmailOrPhone error: ' . $e->getMessage());
      return false;
    }
  }

  // Check if a user exists by email (for registration validation)
  public function findUserByEmail($email){
    try {
      $this->db->query("SELECT * FROM initkey_rid WHERE email = :email");
      $this->db->bind(':email', $email);
      $this->db->single();
      if($this->db->rowCount() > 0){
        return true;
      } else {
        return false;
      }
    } catch (Exception $e) {
      error_log('findUserByEmail error: ' . $e->getMessage());
      return false;
    }
  }

  // Check if a user exists by phone
  public function findUserByPhone($phone) {
    try {
      $this->db->query("SELECT * FROM initkey_rid WHERE phone = :phone");
      $this->db->bind(':phone', $phone);
      $this->db->single();
      if ($this->db->rowCount() > 0) {
        return true;
      } else {
        return false;
      }
    } catch (Exception $e) {
      error_log('findUserByPhone error: ' . $e->getMessage());
      return false;
    }
  }
  
  public function resetPasswordWithToken($email, $token, $new_password)
  {
    try {
      // Get user's stored hashed token and expiry based on email
      $this->db->query('SELECT password_reset_token, reset_token_expires_at FROM initkey_rid WHERE email = :email');
      $this->db->bind(':email', $email);
      $user = $this->db->single();

      // Check if user or token exists
      if (!$user || !$user->password_reset_token) {
        return false;
      }

      // Check if token is expired
      $expiryTime = new DateTime($user->reset_token_expires_at);
      $currentTime = new DateTime();

      if ($currentTime > $expiryTime) {
        return false; // Token expired
      }

      // Verify the token
      if (!password_verify($token, $user->password_reset_token)) {
        return false; // Invalid token
      }

      // Update password and clear the reset token fields
      $this->db->query('UPDATE initkey_rid SET
                       password_hash = :password,
                       password_reset_token = NULL,
                       reset_token_expires_at = NULL
                       WHERE email = :email');

      $this->db->bind(':password', $new_password);
      $this->db->bind(':email', $email);

      return $this->db->execute();
    } catch (Exception $e) {
      error_log('resetPasswordWithToken error: ' . $e->getMessage());
      return false;
    }
  }

  // Find a user by ID
  public function findUserById($userId) {
    try {
      $this->db->query("SELECT * FROM initkey_rid WHERE user_id = :user_id");
      $this->db->bind(':user_id', $userId);
      $row = $this->db->single();
      return $row;
    } catch (Exception $e) {
      error_log('findUserById error: ' . $e->getMessage());
      return false;
    }
  }

  // Find a user by their social ID and provider
  public function findUserBySocialId($socialId, $provider)
  {
    try {
      $this->db->query("SELECT * FROM initkey_rid WHERE social_id = :social_id AND social_provider = :social_provider");
      $this->db->bind(':social_id', $socialId);
      $this->db->bind(':social_provider', $provider);
      $row = $this->db->single();
      return $row;
    } catch (Exception $e) {
      error_log('findUserBySocialId error: ' . $e->getMessage());
      return false;
    }
  }

  public function updateLocation($userId, $locationData)
  {
    try {
      $this->db->query('UPDATE initkey_rid SET
                       address = :address,
                       city = :city,
                       state = :state,
                       lga = :lga,
                       latitude = :latitude,
                       longitude = :longitude
                       WHERE user_id = :user_id');

      $this->db->bind(':address', $locationData['address']);
      $this->db->bind(':city', $locationData['city']);
      $this->db->bind(':state', $locationData['state']);
      $this->db->bind(':lga', $locationData['lga']);
      $this->db->bind(':latitude', $locationData['latitude']);
      $this->db->bind(':longitude', $locationData['longitude']);
      $this->db->bind(':user_id', $userId);
      
      return $this->db->execute();
    } catch (Exception $e) {
      error_log('updateLocation error: ' . $e->getMessage());
      return false;
    }
  }

  // Register a new user with email and password
  public function register($data){
    try {
      $this->db->query('INSERT INTO initkey_rid (name, email, user_id, phone, password_hash, referrer_id, referral_code) 
                       VALUES (:name, :email, :user_id, :phone, :password_hash, :referrer_id, :referral_code)');
      
      $this->db->bind(':name', $data['name']);
      $this->db->bind(':email', $data['email']);
      $this->db->bind(':phone', $data['phone']);
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':password_hash', $data['password']);
      $this->db->bind(':referrer_id', $data['referrer_id'] ?? null);
      $this->db->bind(':referral_code', $data['new_referral_code']);

      return $this->db->execute();
      
    } catch (Exception $e) {
      error_log('User registration failed: ' . $e->getMessage());
      return false;
    }
  }

  // Register a new user from social login data
  public function registerSocialUser($data)
  {
    try {
      $this->db->query('INSERT INTO initkey_rid (user_id, name, email, social_id, social_provider, is_verified, profile_pic_url, referral_code) 
                       VALUES (:user_id, :name, :email, :social_id, :social_provider, :is_verified, :profile_pic_url, :referral_code)');
      
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':name', $data['name']);
      $this->db->bind(':email', $data['email']);
      $this->db->bind(':social_id', $data['social_id']);
      $this->db->bind(':social_provider', $data['social_provider']);
      $this->db->bind(':is_verified', $data['is_verified']);
      $this->db->bind(':profile_pic_url', $data['profile_picture']);
      $this->db->bind(':referral_code', $data['referral_code']);
      
      if($this->db->execute()){
        return $data['user_id'];
      } else {
        return false;
      }
    } catch (Exception $e) {
      error_log('registerSocialUser error: ' . $e->getMessage());
      return false;
    }
  }

  public function updateLastLogin($userId)
  {
      try {
          $this->db->query('UPDATE initkey_rid SET last_login_at = NOW() WHERE user_id = :user_id');
          $this->db->bind(':user_id', $userId);
          return $this->db->execute();
      } catch (Exception $e) {
          error_log('updateLastLogin error: ' . $e->getMessage());
          return false;
      }
  }

  public function linkSocialAccount($userId, $socialId, $provider)
  {
      try {
          $this->db->query('UPDATE initkey_rid SET social_id = :social_id, social_provider = :provider WHERE user_id = :user_id');
          $this->db->bind(':social_id', $socialId);
          $this->db->bind(':provider', $provider);
          $this->db->bind(':user_id', $userId);
          return $this->db->execute();
      } catch (Exception $e) {
          error_log('linkSocialAccount error: ' . $e->getMessage());
          return false;
      }
  }

  public function updateProfilePicture($userId, $pictureUrl)
  {
      try {
          $this->db->query('UPDATE initkey_rid SET profile_pic_url = :picture_url WHERE user_id = :user_id');
          $this->db->bind(':picture_url', $pictureUrl);
          $this->db->bind(':user_id', $userId);
          return $this->db->execute();
      } catch (Exception $e) {
          error_log('updateProfilePicture error: ' . $e->getMessage());
          return false;
      }
  }

  // Add a new role to a user
  public function addRoleToUser($userId, $roleName) {
    try {
      $this->db->query('INSERT INTO UserRoles (user_id, role_name) VALUES (:user_id, :role_name)');
      $this->db->bind(':user_id', $userId);
      $this->db->bind(':role_name', $roleName);
      return $this->db->execute();
    } catch (Exception $e) {
      error_log('addRoleToUser error: ' . $e->getMessage());
      return false;
    }
  }

  // Get all roles for a user
  public function getRolesForUser($userId) {
    try {
      $this->db->query('SELECT role_name FROM UserRoles WHERE user_id = :user_id');
      $this->db->bind(':user_id', $userId);
      return $this->db->resultSet();
    } catch (Exception $e) {
      error_log('getRolesForUser error: ' . $e->getMessage());
      return [];
    }
  }

  // Update user profile
  public function updateProfile($data) {
    try {
      $this->db->query('UPDATE initkey_rid SET
                       name = :name,
                       phone = :phone,
                       location = :location,
                       profile_pic_url = COALESCE(:profile_pic_url, profile_pic_url)
                       WHERE user_id = :user_id');

      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':name', $data['name']);
      $this->db->bind(':phone', $data['phone']);
      $this->db->bind(':location', $data['location']);
      $this->db->bind(':profile_pic_url', $data['profile_pic_url'] ?? null);

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

  // Update user password
  public function updateUserPassword($userId, $new_password){
    try {
      $this->db->query('UPDATE initkey_rid SET password_hash = :password_hash WHERE user_id = :user_id');
      $this->db->bind(':password_hash', $new_password);
      $this->db->bind(':user_id', $userId);
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    } catch (Exception $e) {
      error_log('updateUserPassword error: ' . $e->getMessage());
      return false;
    }
  }

  // Generate password reset token
  public function generatePasswordResetToken($email){
    try {
      $token = bin2hex(random_bytes(16));
      $expiry = date('Y-m-d H:i:s', time() + (30 * 60)); // 30 minutes from now

      $this->db->query('UPDATE initkey_rid SET
                       password_reset_token = :token,
                       reset_token_expires_at = :expiry
                       WHERE email = :email');

      $this->db->bind(':token', password_hash($token, PASSWORD_DEFAULT));
      $this->db->bind(':expiry', $expiry);
      $this->db->bind(':email', $email);

      if($this->db->execute()){
        return $token; // Return plain token for email
      } else {
        return false;
      }
    } catch (Exception $e) {
      error_log('generatePasswordResetToken error: ' . $e->getMessage());
      return false;
    }
  }

  // Find a user by their verification token
  public function findUserByVerificationToken($token)
  {
    try {
      $this->db->query("SELECT * FROM initkey_rid WHERE verification_token = :token AND email_verified_at IS NULL");
      $this->db->bind(':token', $token);
      $row = $this->db->single();
      
      return $row;
    } catch (Exception $e) {
      error_log('findUserByVerificationToken error: ' . $e->getMessage());
      return false;
    }
  }

  // Mark a user's email as verified
  public function markEmailAsVerified($userId)
  {
    try {
      $this->db->query('UPDATE initkey_rid SET email_verified_at = NOW(), verification_token = NULL WHERE user_id = :user_id');
      $this->db->bind(':user_id', $userId);
      
      return $this->db->execute();
    } catch (Exception $e) {
      error_log('markEmailAsVerified error: ' . $e->getMessage());
      return false;
    }
  }

  // Update user access token
  public function updateAccessToken($userId, $token) {
    try {
      $this->db->query('UPDATE initkey_rid SET access_token = :access_token WHERE user_id = :user_id');
      $this->db->bind(':access_token', $token);
      $this->db->bind(':user_id', $userId);
      return $this->db->execute();
    } catch (Exception $e) {
      error_log('updateAccessToken error: ' . $e->getMessage());
      return false;
    }
  }

  public function findUserByReferralCode($referralCode)
  {
    try {
      $this->db->query('SELECT * FROM initkey_rid WHERE referral_code = :referral_code');
      $this->db->bind(':referral_code', $referralCode);
      $row = $this->db->single();
      return $row;
    } catch (Exception $e) {
      error_log('findUserByReferralCode error: ' . $e->getMessage());
      return false;
    }
  }

  public function saveVerificationToken($userId, $token)
  {
    try {
      $this->db->query('UPDATE initkey_rid SET verification_token = :token WHERE user_id = :user_id');
      $this->db->bind(':token', $token);
      $this->db->bind(':user_id', $userId);
      
      return $this->db->execute();
    } catch (Exception $e) {
      error_log('saveVerificationToken error: ' . $e->getMessage());
      return false;
    }
  }

  public function updateToken($user_id, $token, $tag)
  {
    try {
      $this->db->query('UPDATE  initkey_rid SET access_token = :access_token WHERE user_id= :user_id and email = :email');
      $this->db->bind(':user_id', $user_id);
      $this->db->bind(':access_token', $token);
      $this->db->bind(':email', $tag);
      // Execute
      if ($this->db->execute()) {
        return true;
      } else {
        return false;
      }
    } catch (Exception $e) {
      error_log('updateToken error: ' . $e->getMessage());
      return false;
    }
  }

  public function hasRole($userId, $roleName)
  {
    try {
      $this->db->query('SELECT COUNT(*) AS count FROM UserRoles WHERE user_id = :user_id AND role_name = :role_name');
      $this->db->bind(':user_id', $userId);
      $this->db->bind(':role_name', $roleName);
      $result = $this->db->single();
      
      return $result->count > 0;
    } catch (Exception $e) {
      error_log('hasRole error: ' . $e->getMessage());
      return false;
    }
  }

  public function getRolesCountForUser($userId)
  {
    try {
      $this->db->query('SELECT COUNT(*) AS count FROM UserRoles WHERE user_id = :user_id');
      $this->db->bind(':user_id', $userId);
      $result = $this->db->single();
      
      return $result->count;
    } catch (Exception $e) {
      error_log('getRolesCountForUser error: ' . $e->getMessage());
      return 0;
    }
  }

 // In User.php model, add optional parameters:
public function getReferralsForUser($userId, $page = 1, $pageSize = 20)
{
    $offset = ($page - 1) * $pageSize;
    
    // Get total count
    $this->db->query('SELECT COUNT(*) as total FROM initkey_rid WHERE referrer_id = :user_id');
    $this->db->bind(':user_id', $userId);
    $total = $this->db->single()->total;
    
    // Get paginated results
    $this->db->query('SELECT * FROM initkey_rid WHERE referrer_id = :user_id LIMIT :limit OFFSET :offset');
    $this->db->bind(':user_id', $userId);
    $this->db->bind(':limit', $pageSize);
    $this->db->bind(':offset', $offset);
    
    return [
        'data' => $this->db->resultSet(),
        'pagination' => [
            'current_page' => $page,
            'page_size' => $pageSize,
            'total_records' => $total,
            'total_pages' => ceil($total / $pageSize),
            'has_next' => $page < ceil($total / $pageSize),
            'has_prev' => $page > 1
        ]
    ];
}
}