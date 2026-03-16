<?php
  class Shop {
    private $db;
    
    public function __construct(){
      $this->db = new Database;
    }

    // Get All Posts
    public function getPosts(){
      $this->db->query("SELECT *, 
                        posts.id as postId, 
                        initkey.id as userId
                        FROM posts 
                        INNER JOIN initkey 
                        ON posts.user_id = initkey.id
                        ORDER BY posts.created_at DESC;");

      $results = $this->db->resultset();

      return $results;
    }

    // Get Post By ID
    public function getPostById($id){
      $this->db->query("SELECT * FROM posts WHERE id = :id");

      $this->db->bind(':id', $id);
      
      $row = $this->db->single();

      return $row;
    }
    // Get Post By ID
     public function findcable()
{
    try {
        $this->db->query("SELECT * FROM CablePlans");
        $rows = $this->db->resultSet();

        // Check if any rows were returned
        if (!empty($rows)) {
            return $rows;
        } else {
            return null; // Or handle the case when no data is found
        }
    } catch (PDOException $e) {
        // Handle database query errors
        return "Error: " . $e->getMessage();
        // return null; // Or handle the error in an appropriate way
    }
}
     public function finddstv()
{
    try {
        $this->db->query("SELECT * FROM dstv_cable_plans");
        $rows = $this->db->resultSet();

        // Check if any rows were returned
        if (!empty($rows)) {
            return $rows;
        } else {
            return null; // Or handle the case when no data is found
        }
    } catch (PDOException $e) {
        // Handle database query errors
        return "Error: " . $e->getMessage();
        // return null; // Or handle the error in an appropriate way
    }
}
     public function findgotv()
{
    try {
        $this->db->query("SELECT * FROM gotv_cable_plans");
        $rows = $this->db->resultSet();

        // Check if any rows were returned
        if (!empty($rows)) {
            return $rows;
        } else {
            return null; // Or handle the case when no data is found
        }
    } catch (PDOException $e) {
        // Handle database query errors
        return "Error: " . $e->getMessage();
        // return null; // Or handle the error in an appropriate way
    }
}
     public function getAllShops($u)
   {
    try {
        $this->db->query("SELECT * FROM shopdetails WHERE shop_location = :shop_location");
        $this->db->bind(':shop_location', $u);
        $rows = $this->db->resultSet();

        // Check if any rows were returned
        if (!empty($rows)) {
            return $rows;
        } else {
            return null; // Or handle the case when no data is found
        }
    } catch (PDOException $e) {
        // Handle database query errors
        return "Error: " . $e->getMessage();
        // return null; // Or handle the error in an appropriate way
    }
  }
     public function getShop($data, $data2)
   {
     try {
        $this->db->query("SELECT * FROM shopdetails WHERE seller_id = :seller_id AND shop_id = :shop_id");

        $this->db->bind(':seller_id', $data);
        $this->db->bind(':shop_id', $data2);

        $row = $this->db->single();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
     public function getAllShop()
   {
     try {
        $this->db->query("SELECT * FROM shopdetails");

        $row = $this->db->resultSet();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
     public function getShop2($data)
   {
     try {
        $this->db->query("SELECT * FROM shopdetails WHERE seller_id = :seller_id");

        $this->db->bind(':seller_id', $data);
        // $this->db->bind(':shop_id', $data2);

        $row = $this->db->single();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
     public function getShopByshopID($data)
   {
     try {
        $this->db->query("SELECT * FROM shopdetails WHERE seller_id = :seller_id");

        $this->db->bind(':seller_id', $data);

        $row = $this->db->single();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
     public function getUni()
   {
     try {
        $this->db->query("SELECT * FROM uni_data");

       

        $row = $this->db->resultSet();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }
     public function getShopByUni($data)
   {
     try {
        $this->db->query("SELECT * FROM shopdetails WHERE uni_name = :uni_name");

       
    $this->db->bind(':uni_name', $data);
        $row = $this->db->resultSet();

        if (!empty($row)) {
            return $row;
        } else {
            return null; 
        }
        } catch (PDOException $e) {

            return "Error: " . $e->getMessage();

        }

    }


     public function findstartime()
 {
    try {
        $this->db->query("SELECT * FROM startime_cable_plans");
        $rows = $this->db->resultSet();

        // Check if any rows were returned
        if (!empty($rows)) {
            return $rows;
        } else {
            
            return null; // Or handle the case when no data is found
        }
    } catch (PDOException $e) {
        // Handle database query errors
        return "Error: " . $e->getMessage();
        // return null; // Or handle the error in an appropriate way
    }
  }
 

    // Add Post
    public function createShop($data) {
        //  print_r(json_encode($data));
        //     exit;
        // Prepare Query
        $this->db->query('UPDATE shopdetails SET 
            shop_name = :shop_name, 
            shop_desc = :shop_desc, 
            shop_whatsapp_link = :shop_whatsapp_link, 
            service_offered = :service_offered, 
            shop_image_url = :shop_image_url,
            uni_name = :uni_name,
            shop_location = :shop_location,
            seller_id = :seller_id,
            shop_tag = :shop_tag
            WHERE shop_id = :shop_id');
        
        // Bind Values
        $this->db->bind(':shop_name', $data['shop_name']);
        $this->db->bind(':shop_desc', $data['shop_desc']);
        $this->db->bind(':shop_whatsapp_link', $data['shop_whatsapp_link']);
        $this->db->bind(':service_offered', $data['service_offered']);
        $this->db->bind(':shop_image_url', $data['shop_image_url']);
        $this->db->bind(':uni_name', $data['shop_location']);
        $this->db->bind(':shop_location', $data['shop_location']);
        $this->db->bind(':seller_id', $data['seller_id']);
        $this->db->bind(':shop_tag', $data['shop_tag']);
        $this->db->bind(':shop_id', $data['shop_id']);
        
        // Execute
        if($this->db->execute()){
            return true;
        } else {
            return false;
        }
    }
    public function createShop22($data) {
        // print_r(json_encode($data));exit;
         $this->db->query('UPDATE shopdetails SET 
            shop_name = :shop_name, 
            shop_desc = :shop_desc, 
            shop_whatsapp_link = :shop_whatsapp_link, 
            service_offered = :service_offered, 
            shop_image_url = :shop_image_url,
            uni_name = :uni_name,
            shop_location = :shop_location,
            shop_tag = :shop_tag
        WHERE
            seller_id = :seller_id AND
            shop_id = :shop_id');
    
   $this->db->bind(':shop_name', $data['shop_name']);
$this->db->bind(':shop_desc', $data['shop_desc']);
$this->db->bind(':shop_whatsapp_link', $data['shop_whatsapp_link']);
$this->db->bind(':service_offered', $data['service_offered']);
$this->db->bind(':shop_image_url', $data['shop_image_url']);
$this->db->bind(':uni_name', $data['shop_location']); // Fixed overlap
$this->db->bind(':shop_location', $data['shop_location']);
$this->db->bind(':seller_id', $data['seller_id']);
 $this->db->bind(':shop_tag', $data['shop_tag']);
$this->db->bind(':shop_id', $data['shop_id']);
    
    // Execute
    if ($this->db->execute()) {
        return true;
    } else {
        return false;
    
}

    }
    public function activateShop($data) {
        // Prepare Query
        $this->db->query('INSERT INTO shopdetails SET 
            shop_id = :shop_id, 
            shop_plan = :shop_plan, 
            plan_price = :plan_price, 
            expiry_date = :expiry_date, 
            seller_id = :seller_id');
        
        // Bind Values
        $this->db->bind(':shop_id', $data['shop_id']);
        $this->db->bind(':shop_plan', $data['shop_plan']);
        $this->db->bind(':plan_price', $data['plan_price']);
        $this->db->bind(':expiry_date', $data['expiry_date']);
        $this->db->bind(':seller_id', $data['seller_id']);
        
        // Execute
        if($this->db->execute()){
            $this->db->query('UPDATE initkey_rid SET 
            shop_id = :shop_id, 
            shop_plan = :shop_plan, 
            plan_price = :plan_price, 
            shop_status = :shop_status, 
            expiry_date = :expiry_date
            WHERE user_id = :user_id');
        
        // Bind Values
        $this->db->bind(':shop_id', $data['shop_id']);
        $this->db->bind(':shop_plan', $data['shop_plan']);
        $this->db->bind(':plan_price', $data['plan_price']);
        $this->db->bind(':shop_status', 1);
        $this->db->bind(':expiry_date', $data['expiry_date']);
        $this->db->bind(':user_id', $data['seller_id']);
        
        // Execute
        if($this->db->execute()){
            return true;
        } else {
            return false;
        }
        } else {
            return false;
        }
    }
    
    public function saveRequest($data) {
        // Prepare Query
        $this->db->query('INSERT INTO product_requests SET 
            product_name = :product_name, 
            price = :price, 
            number = :number, 
            img = :img, 
            `desc` = :desc,
            request_id = :request_id');
        
        // Bind Values
        $this->db->bind(':product_name', $data['product_name']);
        $this->db->bind(':price', $data['price']);
        $this->db->bind(':number', $data['number']);
        $this->db->bind(':img', $data['img']);
        $this->db->bind(':desc', $data['desc']);
        $this->db->bind(':request_id', $data['request_id']);
        
        // Execute
        if($this->db->execute()){
            return true;
        } else {
            return false;
        }
    }
    
    public function updateWallet($data){
      // Prepare Query
      $this->db->query('INSERT INTO transactions (username, userid, reference, amount, status,email,tr_type) 
      VALUES (:username, :userid, :reference, :amount, :status, :email, :tr_type)');

      // Bind Values
      $this->db->bind(':username', $data['username']);
      $this->db->bind(':userid', $data['user_id']);
      $this->db->bind(':reference', $data['reference']);
      $this->db->bind(':amount', $data['amount']);
      $this->db->bind(':status', $data['status']);
      $this->db->bind(':email', $data['email']);
      $this->db->bind(':tr_type', 'deposit');
      
      //Execute
      if($this->db->execute()){
       if ( $this->updateacc($data)) {
         return true;
       }else {
        return false;
       }
      } else {
        return false;
      }
    }
    public function updateWallet2($data){
      // Prepare Query
      $this->db->query('INSERT INTO transactions (username, userid, reference, amount, status,email,tr_type) 
      VALUES (:username, :userid, :reference, :amount, :status, :email, :tr_type)');

      // Bind Values
      $this->db->bind(':username', $data['username']);
      $this->db->bind(':userid', $data['user_id']);
      $this->db->bind(':reference', $data['reference']);
      $this->db->bind(':amount', $data['amount']);
      $this->db->bind(':status', $data['status']);
      $this->db->bind(':email', $data['email']);
      $this->db->bind(':tr_type', 'deposit');
      
      //Execute
      if($this->db->execute()){
       if ( $this->updateacc($data)) {
         return true;
       }else {
        return false;
       }
      } else {
        return false;
      }
    }
     public function buyData($data){
      // Prepare Query
      $this->db->query('INSERT INTO transactions (username, userid, reference,  amount, status, email, tr_type) 
      VALUES (:username, :userid, :reference, :amount, :status, :email, :tr_type)');

      // Bind Values
      $this->db->bind(':username' ,$_SESSION['user_name']);
      $this->db->bind(':userid', $data['user_id']);
      $this->db->bind(':reference', $data['ref']);
      $this->db->bind(':amount', $data['amount']);
      $this->db->bind(':status', 'successful');
      $this->db->bind(':email', $_SESSION['user_email']);
    //   $this->db->bind(':network_id', $data['network_id']);
    //   $this->db->bind(':phone', $data['phone']);
    //   $this->db->bind(':plan_id', $data['plan_id']);
      $this->db->bind(':tr_type', 'data');
       
      //Execute
      if($this->db->execute()){
       if ( $this->reduceacc($data)) {
         return true;
       }else {
        return false;
       }
      } else {
        return false;
      }
    }
 public function buyAirtime($data){
      // Prepare Query
      $this->db->query('INSERT INTO transactions (username, userid,reference, amount, status, email, tr_type) 
      VALUES (:username, :userid, :reference, :amount, :status, :email, :tr_type)');

      // Bind Values
      $this->db->bind(':username', $data['username']);
      $this->db->bind(':userid', $data['user_id']);
      $this->db->bind(':reference', $data['ref']);
      $this->db->bind(':amount', $data['amount']);
      $this->db->bind(':status', 'successful');
      $this->db->bind(':email', $data['email']);
    //   $this->db->bind(':network_id', $data['network']);
    //   $this->db->bind(':phone', $data['phone']);
      // $this->db->bind(':plan_id', $data['plan_id']);
      $this->db->bind(':tr_type', 'airtime');
      
      //Execute
      if($this->db->execute()){
       if ($this->reduceacc($data)) {
         return true;
       }else {
        return false;
       }
      } else {
        return false;
      }
    }

    // Update Post
    public function updatePost($data){
      // Prepare Query
      $this->db->query('UPDATE posts SET title = :title, body = :body WHERE id = :id');

      // Bind Values
      $this->db->bind(':id', $data['id']);
      $this->db->bind(':title', $data['title']);
      $this->db->bind(':body', $data['body']);
      
      //Execute
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    }
    public function updateacc($data){
      // Prepare Query
      $this->db->query('SELECT * FROM user_account WHERE user_id = :user_id');
      $this->db->bind(':user_id', $data['user_id']);
      $row = $this->db->single();

      $this->db->query('UPDATE user_account SET acountbalance = :acountbalance WHERE user_id = :user_id');

      // Bind Values
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':acountbalance', $data['amount'] + $row->acountbalance);
      
      
      //Execute
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    }
    public function reduceacc($data){
      // Prepare Query
      $this->db->query('SELECT * FROM user_account WHERE user_id = :user_id');
      $this->db->bind(':user_id', $data['user_id']);
      $row = $this->db->single();

      $this->db->query('UPDATE user_account SET acountbalance = :acountbalance WHERE user_id = :user_id');

      // Bind Values
      $this->db->bind(':user_id', $data['user_id']);
      $this->db->bind(':acountbalance', $row->acountbalance - $data['amount']);


      $_SESSION['accb'] = $row->acountbalance - $data['amount'];
      
      
      //Execute
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    }

    // Delete Post
    public function deletePost($id){
      // Prepare Query
      $this->db->query('DELETE FROM posts WHERE id = :id');

      // Bind Values
      $this->db->bind(':id', $id);
      
      //Execute
      if($this->db->execute()){
        return true;
      } else {
        return false;
      }
    }
  }