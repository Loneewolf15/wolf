<?php 
class Report {
    private $db;

    public function __construct(){
        $this->db = new Database;
    }




    //Get all submule 
    public function getVpayToken(){
        $id = 1;
        $this->db->query("SELECT * FROM adminpanel WHERE id = :id	");
        $this->db->bind(':id', $id);
        $row = $this->db->single();
        // Check roow
        return $row->VpayToken;
         
      
    }
    public function getUrl(){
        $id = 1;
        $this->db->query("SELECT * FROM adminpanel WHERE id = :id	");
        $this->db->bind(':id', $id);
        $row = $this->db->single();
        // Check roow
        return $row->watsapp;
         
      
    }
    

      //Get all submule 
      public function getBatchByID($catID){
        $this->db->query("SELECT * FROM category WHERE catID = :catID	");
        $this->db->bind(':catID', $catID);
          
        $row = $this->db->single();
        // Check roow
        return $row;
         
      
    }

    public function findUserByEmail($email)
    {
        $this->db->query("SELECT * FROM initkeyrid WHERE  email = :email");

        // Bind Values
        $this->db->bind(':email', $email);

        $row = $this->db->single();

        // Check roow
        if ($this->db->rowCount() > 0) {
            return true;
        } else {
            return false;
        }


      }


    //Get all submule 
    public function findmtndata(){
      $this->db->query("SELECT * FROM mtn_data_plans");
    
      $row = $this->db->resultSet();
      // Check roow
      return $row;
       
    
  }
    public function findetisalatdata(){
      $this->db->query("SELECT * FROM 9mobile_data_plans");

    
      $row = $this->db->resultSet();
      // Check roow
      return $row;
       
    
  }
    public function findairteldata(){
      $this->db->query("SELECT * FROM airtel_data_plans");
    
      $row = $this->db->resultSet();
      // Check roow
      return $row;
       
    
  }
    public function getNotifications(){
      $this->db->query("SELECT * FROM notification ORDER BY id desc");
    
      $rows = $this->db->resultSet();
       if ($rows) {
            return $rows;
        } else {
              $res = [
        'status' => "false",
        'message' => 'no notification',
        
        ];
        
        return $res;
        

        }
       
    
  }
 
    public function setNotifications(){
       $this->db->query('INSERT INTO  notification (headers, text,img, date) VALUES( :headers, :text,:img, :date)');
      // Bind Valuesrid
       $this->db->bind(':headers', $datax['headers']);
      $this->db->bind(':text', $datax['text']);
      $this->db->bind(':img', $datax['img']);
       date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', date('Y-m-d H:i:s'));
    
      if ($this->db->execute())
      {
          return true;
      }else 
      {
          return false;
      }
       
    
  }
 public function findglodata()
{
    try {
        $this->db->query("SELECT * FROM glo_data_plans");
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
 public function getAllpendingTransaction()
{
    try {
        $this->db->query("SELECT * FROM pending_transaction_history WHERE transaction_status = :transaction_status ");
         $this->db->bind(':transaction_status', 'pending');
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

  public function getAvailableCourses($catID)
  {
    $this->db->query("SELECT * FROM courses_final WHERE catID = :catID order by code asc");

    // print_r($catID);
    // exit;

    // Bind Values
    $this->db->bind(':catID', $catID);

    $row = $this->db->resultSet();
    $roC = $this->db->rowCount();
    // Check row
    return $row;
  }
  public function findUserByTagName($tagname)
  {
      $this->db->query("SELECT * FROM initkeyrid WHERE  user_tag = :user_tag");

      // Bind Values
      $this->db->bind(':user_tag', $tagname);

      $row = $this->db->single();

      // Check roow
      if ($this->db->rowCount() > 0) {
          return $row;
      } else {
          return false;
      }
  }
  public function findSubUserByTagNamex($tagname)
  {
      $this->db->query("SELECT * FROM sub_initkeyrid WHERE  user_tag = :user_tag");

      // Bind Values
      $this->db->bind(':user_tag', $tagname);

      $row = $this->db->single();

      // Check roow
      if ($this->db->rowCount() > 0) {
          
          return $row;
      } else {
          return false;
      }
  }
  public function findUserMainAcc($tagname)
  {
      $this->db->query("SELECT * FROM initkeyrid WHERE veluxite_id = :veluxite_id");

      // Bind Values
      $this->db->bind(':veluxite_id', $tagname);

      $row = $this->db->single();

      // Check roow
      if ($this->db->rowCount() > 0) {
          return $row;
      } else {
          return false;
      }
  }
  public function findUserByTagNamex($tagname)
  {
      $this->db->query("SELECT * FROM initkeyrid WHERE  user_tag = :user_tag");

      // Bind Values
      $this->db->bind(':user_tag', $tagname);

      $row = $this->db->single();

      // Check roow
      if ($this->db->rowCount() > 0) {
          return true;
      } else {
          return false;
      }
  }
  public function findSubUserByTagName($tagname)
  {
      $this->db->query("SELECT * FROM sub_initkeyrid WHERE  user_tag = :user_tag");

      // Bind Values
      $this->db->bind(':user_tag', $tagname);

      $row = $this->db->single();

      // Check roow
      if ($this->db->rowCount() > 0) {
          return true;
      } else {
          return false;
      }
  }
  public function  inAppTransfer($datax)
  {
// print_r(json_encode($datax));
// exit;

      $this->db->query('INSERT INTO  transaction_history (sender_tagname,sender_name, receiver_tagname, receiver_name,sender_id,receiver_id, transaction_status, amount, transaction_id, date) VALUES( :sender_tagname,:sender_name, :receiver_tagname,:receiver_name,:sender_id, :receiver_id, :transaction_status, :amount, :transaction_id, :date)');
      // Bind Valuesrid
      $this->db->bind(':sender_tagname', $datax['s_tag']);
      $this->db->bind(':sender_name', $datax['s_name']);
      $this->db->bind(':receiver_tagname', $datax['r_tag']);
      $this->db->bind(':receiver_name', $datax['r_name']);
      $this->db->bind(':sender_id', $datax['s_id']);
      $this->db->bind(':receiver_id', $datax['r_id']);
      $this->db->bind(':transaction_status', $datax['tr_status']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':transaction_id', $datax['tr_id']);
      // Set the PHP timezone to GMT
      date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', date('Y-m-d H:i:s'));
    
      //$this->db->execute();


      if ($this->db->execute()) {
          
          // Insert into all_transactions table
       $this->db->query('INSERT INTO  all_transactions (sender_tagname,sender_name, receiver_tagname, receiver_name,sender_id,receiver_id, transaction_status, amount, transaction_id, transaction_type, date) VALUES( :sender_tagname,:sender_name, :receiver_tagname,:receiver_name,:sender_id, :receiver_id, :transaction_status, :amount, :transaction_id, :transaction_type, :date)');
      // Bind Valuesrid
      $this->db->bind(':sender_tagname', $datax['s_tag']);
      $this->db->bind(':sender_name', $datax['s_name']);
      $this->db->bind(':receiver_tagname', $datax['r_tag']);
      $this->db->bind(':receiver_name', $datax['r_name']);
      $this->db->bind(':sender_id', $datax['s_id']);
      $this->db->bind(':receiver_id', $datax['r_id']);
      $this->db->bind(':transaction_status', $datax['tr_status']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':transaction_id', $datax['tr_id']);
      $this->db->bind(':transaction_type', 'inapp');
       // Set the PHP timezone to GMT
    date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
      $this->db->bind(':date', date('Y-m-d H:i:s'));
    
      //$this->db->execute();


      if ($this->db->execute()) {
        //print_r($datax)
         // return true;
          return true;
      } else {
          return false;
      }



      } else {
        //   return false;
        if (!$this->db->execute()) {
    // Log or display the error
   echo "error here";
    return false;
}
      }




  }
  public function  inAppTransfer2($datax)
  {
// print_r(json_encode($datax));
// exit;

      $this->db->query('INSERT INTO  transaction_history (sender_tagname,sender_name, receiver_tagname, receiver_name,sender_id,receiver_id, transaction_status, amount, transaction_id, date) VALUES( :sender_tagname,:sender_name, :receiver_tagname,:receiver_name,:sender_id, :receiver_id, :transaction_status, :amount, :transaction_id, :date)');
      // Bind Valuesrid
      $this->db->bind(':sender_tagname', $datax['s_tag']);
      $this->db->bind(':sender_name', $datax['s_name']);
      $this->db->bind(':receiver_tagname', $datax['r_tag']);
      $this->db->bind(':receiver_name', $datax['r_name']);
      $this->db->bind(':sender_id', $datax['s_id']);
      $this->db->bind(':receiver_id', $datax['r_id']);
      $this->db->bind(':transaction_status', $datax['tr_status']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':transaction_id', $datax['tr_id']);
      // Set the PHP timezone to GMT
      date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', $datax['date']);
    
      //$this->db->execute();


      if ($this->db->execute()) {
          
          // Insert into all_transactions table
       $this->db->query('INSERT INTO  all_transactions (sender_tagname,sender_name, receiver_tagname, receiver_name,sender_id,receiver_id, transaction_status, amount, transaction_id, transaction_type, date) VALUES( :sender_tagname,:sender_name, :receiver_tagname,:receiver_name,:sender_id, :receiver_id, :transaction_status, :amount, :transaction_id, :transaction_type, :date)');
      // Bind Valuesrid
      $this->db->bind(':sender_tagname', $datax['s_tag']);
      $this->db->bind(':sender_name', $datax['s_name']);
      $this->db->bind(':receiver_tagname', $datax['r_tag']);
      $this->db->bind(':receiver_name', $datax['r_name']);
      $this->db->bind(':sender_id', $datax['s_id']);
      $this->db->bind(':receiver_id', $datax['r_id']);
      $this->db->bind(':transaction_status', $datax['tr_status']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':transaction_id', $datax['tr_id']);
      $this->db->bind(':transaction_type', 'inapp');
       // Set the PHP timezone to GMT
    date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
       $this->db->bind(':date', $datax['date']);
    
      //$this->db->execute();


      if ($this->db->execute()) {
        //print_r($datax)
         // return true;
          return true;
      } else {
          return false;
      }



      } else {
        //   return false;
        if (!$this->db->execute()) {
    // Log or display the error
   echo "error here";
    return false;
}
      }




  }
  public function  pendinginAppTransfer($datax)
  {
// print_r(json_encode($datax));
// exit;

      $this->db->query('INSERT INTO  pending_transaction_history (sender_tagname,sender_name, receiver_tagname, receiver_name,sender_id,receiver_id, transaction_status, targettime, amount, transaction_id, date) VALUES( :sender_tagname,:sender_name, :receiver_tagname,:receiver_name,:sender_id, :receiver_id, :transaction_status, :targettime, :amount, :transaction_id, :date)');
      // Bind Valuesrid
      $this->db->bind(':sender_tagname', $datax['s_tag']);
      $this->db->bind(':sender_name', $datax['s_name']);
      $this->db->bind(':receiver_tagname', $datax['r_tag']);
      $this->db->bind(':receiver_name', $datax['r_name']);
      $this->db->bind(':sender_id', $datax['s_id']);
      $this->db->bind(':receiver_id', $datax['r_id']);
      $this->db->bind(':transaction_status', 'pending');
      $this->db->bind(':targettime', $datax['targettime']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':transaction_id', $datax['tr_id']);
      // Set the PHP timezone to GMT
      date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', date('Y-m-d H:i:s'));
    
      //$this->db->execute();


      if ($this->db->execute()) {
          
       return true;

      } else {
        //   return false;
        if (!$this->db->execute()) {
    // Log or display the error
   echo "error here";
    return false;
}
      }




  }
 public function pendingupdate($datax)
{
    $this->db->query('UPDATE pending_transaction_history 
                      SET transaction_status = :transaction_status 
                      WHERE transaction_id = :transaction_id');

    // Bind Values
    $this->db->bind(':transaction_status', 'successful');
    $this->db->bind(':transaction_id', $datax['tr_id']);

    // Set the PHP timezone to GMT
    date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
    // $this->db->bind(':date', date('Y-m-d H:i:s'));

    if ($this->db->execute()) {
        return true;
    } else {
        // Log or display the error
        echo "Error updating transaction status";
        return false;
    }
}

  public function  inAppTransferSub($datax)
  {
// print_r(json_encode($datax));
// exit;

      $this->db->query('INSERT INTO  sub_transaction_history (sender_tagname,sender_name, receiver_tagname, receiver_name,sender_id,receiver_id, transaction_status, amount, transaction_id) VALUES( :sender_tagname,:sender_name, :receiver_tagname,:receiver_name,:sender_id, :receiver_id, :transaction_status, :amount, :transaction_id)');
      // Bind Valuesrid
      $this->db->bind(':sender_tagname', $datax['s_tag']);
      $this->db->bind(':sender_name', $datax['s_name']);
      $this->db->bind(':receiver_tagname', $datax['r_tag']);
      $this->db->bind(':receiver_name', $datax['r_name']);
      $this->db->bind(':sender_id', $datax['s_id']);
      $this->db->bind(':receiver_id', $datax['r_id']);
      $this->db->bind(':transaction_status', $datax['tr_status']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':transaction_id', $datax['tr_id']);
    
      //$this->db->execute();


      if ($this->db->execute()) {
          return true;
      } else {
          return false;
      }





  }
  public function  deposite($datax)
  {

      $this->db->query('INSERT INTO  deposite_ (user_tag,full_name, veluxite_id, transaction_status, amount, transaction_id, transaction_ref, date) VALUES( :user_tag,:full_name, :veluxite_id, :transaction_status, :amount, :transaction_id, :transaction_ref, :date)');
      // Bind Valuesrid
      $this->db->bind(':user_tag', $datax['tagname']);
      $this->db->bind(':full_name', $datax['fulname']);
      $this->db->bind(':veluxite_id', $datax['s_id']);
      $this->db->bind(':transaction_status', $datax['tr_status']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':transaction_id', $datax['tr_id']);
      $this->db->bind(':transaction_ref', $datax['t_ref']);
      // Set the PHP timezone to GMT
      date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', date('Y-m-d H:i:s'));
    
      //$this->db->execute();


        if ($this->db->execute()) {
          
          // Insert into all_transactions table
       $this->db->query('INSERT INTO  all_transactions (user_tag,full_name, veluxite_id, transaction_status, amount, transaction_id, transaction_ref, transaction_type, date) VALUES( :user_tag,:full_name, :veluxite_id, :transaction_status, :amount, :transaction_id, :transaction_ref, :transaction_type, :date)');
      // Bind Valuesrid
      $this->db->bind(':user_tag', $datax['tagname']);
      $this->db->bind(':full_name', $datax['fulname']);
      $this->db->bind(':veluxite_id', $datax['s_id']);
      $this->db->bind(':transaction_status', $datax['tr_status']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':transaction_id', $datax['tr_id']);
      $this->db->bind(':transaction_ref', $datax['t_ref']);
      $this->db->bind(':transaction_type', 'credit');
       // Set the PHP timezone to GMT
    date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
      $this->db->bind(':date', date('Y-m-d H:i:s'));
    
      //$this->db->execute();


      if ($this->db->execute()) {
        //print_r($datax)
         // return true;
          return true;
      } else {
          return false;
      }



      } else {
        //   return false;
        if (!$this->db->execute()) {
    // Log or display the error
   echo "error here";
    return false;
}
      }




  }
  
  public function  withdrawFunds($datax)
  {


      $this->db->query('INSERT INTO  withdrawal_record (fullname, veluxite_id,bank_code,bankname,accountname,amount, tr_id, date) VALUES( :fullname, :veluxite_id,:bank_code,:bankname,:accountname, :amount, :tr_id, :date)');
      // Bind Valuesrid
      $this->db->bind(':fullname', $datax['fullname']);
      $this->db->bind(':veluxite_id', $datax['veluxite_id']);
      $this->db->bind(':bank_code', $datax['bank_code']);
      $this->db->bind(':bankname', $datax['bank_name']);
      $this->db->bind(':accountname', $datax['accname']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':tr_id', $datax['tr_id']);
      // Set the PHP timezone to GMT
      date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', date('Y-m-d H:i:s'));
      if ($this->db->execute()) {
          
            // Insert into all_transactions table
       $this->db->query('INSERT INTO  all_transactions (fullname, veluxite_id,bank_code,bankname,accountname,amount, tr_id, transaction_type, date) VALUES( :fullname, :veluxite_id,:bank_code,:bankname,:accountname, :amount, :tr_id, :transaction_type, :date)');
      // Bind Valuesrid
       $this->db->bind(':fullname', $datax['fullname']);
      $this->db->bind(':veluxite_id', $datax['veluxite_id']);
      $this->db->bind(':bank_code', $datax['bank_code']);
      $this->db->bind(':bankname', $datax['bank_name']);
      $this->db->bind(':accountname', $datax['accname']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':tr_id', $datax['tr_id']);
      $this->db->bind(':transaction_type', 'withdrawal');
       // Set the PHP timezone to GMT
      date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', date('Y-m-d H:i:s'));
          
            if ($this->db->execute()) {
           $this->db->query('SELECT * FROM user_accounts WHERE  usertag = :usertag and  userid = :userid ');
      // Bind Valuesrid
      $this->db->bind(':usertag', $datax['Tag']);
      $this->db->bind(':userid', $datax['veluxite_id']);
      $row = $this->db->single();

      // Check roow
      if ($this->db->rowCount() > 0) {
        if ($row->accountbalance > $datax['amount']) {
           $newFund = $row->accountbalance - $datax['amount'];
             $this->db->query('UPDATE user_accounts set accountbalance = :accountbalance WHERE  usertag = :usertag and  userid = :userid');

        $this->db->bind(':usertag', $datax['Tag']);
        $this->db->bind(':userid', $datax['veluxite_id']);
        $this->db->bind(':accountbalance', $newFund);
        if ($this->db->execute()){
            return true;
        
      } else {
          return false;
      }
          
          
       
      } else {
          return false;
      }
}}}

}
  
  public function  accountUpdate($datax)
  {

      $this->db->query('SELECT * FROM user_accounts WHERE  usertag = :usertag and  userid = :userid ');
      // Bind Valuesrid
      $this->db->bind(':usertag', $datax['s_tag']);
      $this->db->bind(':userid', $datax['s_id']);
      $row = $this->db->single();

      // Check roow
      if ($this->db->rowCount() > 0) {
        if ($row->accountbalance > $datax['amount']) {
           $newFund = $row->accountbalance - $datax['amount'];
             $this->db->query('UPDATE user_accounts set accountbalance = :accountbalance WHERE  usertag = :usertag and  userid = :userid');

        $this->db->bind(':usertag', $datax['s_tag']);
        $this->db->bind(':userid', $datax['s_id']);
        $this->db->bind(':accountbalance', $newFund);
        if ($this->db->execute()) {
                    $this->db->query('SELECT * FROM user_accounts WHERE  usertag = :usertag and  userid = :userid ');
        // Bind Valuesrid
        $this->db->bind(':usertag', $datax['r_tag']);
        $this->db->bind(':userid', $datax['r_id']);
                $rowx = $this->db->single();
        if ($this->db->rowCount() > 0) {
        
            $newFundx = $rowx->accountbalance + $datax['amount'];
            $this->db->query('UPDATE user_accounts set accountbalance = :accountbalance WHERE  usertag = :usertag and  userid = :userid');

            $this->db->bind(':usertag', $datax['r_tag']);
            $this->db->bind(':userid', $datax['r_id']);
            $this->db->bind(':accountbalance', $newFundx);
            if ($this->db->execute()) {
                return true;
            } else {
                $res = [
                    'status' => 401,
                    'message' => 'failed',
                ];
                print_r(json_encode($res));
                exit;
            }
        }else {
            return false;
        }

        } else {
            $res = [
                'status' => 401,
                'message' => 'failed',
              ];
              print_r(json_encode($res));
              exit;
        }

      } else {
        return false;
      }
        }else {
            $res = [
                'status' => 401,
                'message' => 'not enough Balance',
              ];
              print_r(json_encode($res));
              exit;
        }
       
       





  }
  public function  accountUpdateSub($datax)
  {
                    $this->db->query('SELECT * FROM sub_initkeyrid WHERE  user_tag = :user_tag and  veluxite_id = :veluxite_id ');
        // Bind Valuesrid
        $this->db->bind(':user_tag', $datax['r_tag']);
        $this->db->bind(':veluxite_id', $datax['r_id']);
                $rowx = $this->db->single();
        if ($this->db->rowCount() > 0) {
        
            $newFundx = $rowx->acc_balance + $datax['amount'];
            $this->db->query('UPDATE sub_initkeyrid set acc_balance = :acc_balance WHERE  user_tag = :user_tag and  veluxite_id = :veluxite_id');

            $this->db->bind(':user_tag', $datax['r_tag']);
            $this->db->bind(':veluxite_id', $datax['r_id']);
            $this->db->bind(':acc_balance', $newFundx);
            if ($this->db->execute()) {
                return true;
            } else {
                $res = [
                    'status' => 401,
                    'message' => 'failed',
                ];
                print_r(json_encode($res));
                exit;
            }
        }else {
            return false;
        }

        

    
       
       





  }
  
  public function creditUser($datax)
{
    $this->db->query('SELECT * FROM user_accounts WHERE usertag = :usertag AND userid = :userid');
    $this->db->bind(':usertag', $datax['tagname']);
    $this->db->bind(':userid', $datax['s_id']);
    $row = $this->db->single();

    if ($this->db->rowCount() > 0) {
        $newFund = $row->accountbalance + $datax['amount'];
        $this->db->query('UPDATE user_accounts SET accountbalance = :accountbalance WHERE usertag = :usertag AND userid = :userid');
        $this->db->bind(':usertag', $datax['tagname']);
        $this->db->bind(':userid', $datax['s_id']);
        $this->db->bind(':accountbalance', $newFund);
        if ($this->db->execute()) {
            return true;
        } else {
            return false;
        }
    } else {
        $res = [
            'status' => 401,
            'message' => 'failed: user account not found'
        ];
        print_r(json_encode($res));
        exit;
    }
}


  public function  creditUsery($datax)
  {
    //   print_r(json_encode($datax));
      $this->db->query('SELECT * FROM user_accounts WHERE  usertag = :usertag and  userid = :userid ');
      // Bind Valuesrid
      $this->db->bind(':usertag', $datax['tagname']);
      $this->db->bind(':userid', $datax['s_id']);
      $row = $this->db->single();

      // Check roow
      if ($this->db->rowCount() > 0) {
        if ($row->accountbalance) {
           $newFund = $row->accountbalance + $datax['amount'];
             $this->db->query('UPDATE user_accounts set accountbalance = :accountbalance WHERE  usertag = :usertag and  userid = :userid');

        $this->db->bind(':usertag', $datax['tagname']);
        $this->db->bind(':userid', $datax['s_id']);
        $this->db->bind(':accountbalance', $newFund);
        if ($this->db->execute()) {
                   
                return true;
            } else {
              
            return false;
        }

     
        }else {
            $res = [
                'status' => 401,
                'message' => 'not enough Balance',
              ];
              print_r(json_encode($res));
              exit;
        }
       
      }else {
                $res = [
                    'status' => 401,
                    'message' => 'failed',
                ];
                print_r(json_encode($res));
                exit;
            }





  }
    

  public function  accountAdd($datax)
  {


      $this->db->query('INSERT INTO  user_accounts (usertag, userid) VALUES( :usertag, :userid)');
      // Bind Valuesrid
      $this->db->bind(':usertag', $datax['usertag']);
      $this->db->bind(':userid', $datax['userid']);
    //   $this->db->bind(':transaction', $datax['transaction']);
    //   $this->db->bind(':accountbalance', $datax['accountbalance']);
      if ($this->db->execute()) {
          return true;
      } else {
          return false;
      }
  }
  public function  checkAccount($tag, $userId, $amount)
  {


      $this->db->query('SELECT * FROM user_accounts WHERE   usertag = :usertag and  userid = :userid ');
      // Bind Valuesrid
      $this->db->bind(':usertag', $tag);
      $this->db->bind(':userid', $userId);
         $row = $this->db->single();
      if ($this->db->rowCount() > 0) {
        if ($row->accountbalance > $amount) {
            return true;
      } else {
          return false;
      }
      
      }else{
          return false; 
      }
      
      





  }
   public function  payVeluxiteAdmin($datax)
  {
    $this->db->query('SELECT * FROM user_accounts WHERE  usertag = :usertag and  userid = :userid ');
    // Bind Valuesrid
    $this->db->bind(':usertag', $datax['r_tag']);
    $this->db->bind(':userid', $datax['r_id']);
    $row = $this->db->single();

    // Check roow
            if ($this->db->rowCount() > 0) {
            $newFund = $row->accountbalance + $datax['amount'];
            $this->db->query('UPDATE user_accounts set accountbalance = :accountbalance WHERE  usertag = :usertag and  userid = :userid');

            $this->db->bind(':usertag', $datax['r_tag']);
            $this->db->bind(':userid', $datax['r_id']);
            $this->db->bind(':accountbalance', $newFund);
            if ($this->db->execute()) {
                return true;
            } else {
                $res = [
                    'status' => 401,
                    'message' => 'failed',
                ];
                print_r(json_encode($res));
                exit;
            }
        }else {
            return false;
        }


}
   public function  creditAdmin($datax)
  {
    //   print_r(json_encode($datax));
    $this->db->query('SELECT * FROM user_accounts WHERE  usertag = :usertag and  userid = :userid ');
    // Bind Valuesrid
    $this->db->bind(':usertag', $datax['tagname']);
    $this->db->bind(':userid', $datax['s_id']);
    $row = $this->db->single();

    // Check roow
            if ($this->db->rowCount() > 0) {
            $newFund = $row->accountbalance + $datax['amount'];
            $this->db->query('UPDATE user_accounts set accountbalance = :accountbalance WHERE  usertag = :usertag and  userid = :userid');

            $this->db->bind(':usertag', $datax['tagname']);
            $this->db->bind(':userid', $datax['s_id']);
            $this->db->bind(':accountbalance', $newFund);
            if ($this->db->execute()) {
                return true;
            } else {
                $res = [
                    'status' => 401,
                    'message' => 'failed',
                ];
                print_r(json_encode($res));
                exit;
            }
        }else {
            return false;
        }


}
   public function  payVeluxiteAdmin2($datax)
  {
    $this->db->query('SELECT * FROM user_accounts WHERE  usertag = :usertag and  userid = :userid ');
    // Bind Valuesrid
    $this->db->bind(':usertag', $datax['r_tag']);
    $this->db->bind(':userid', $datax['r_id']);
    $row = $this->db->single();

    // Check roow
            if ($this->db->rowCount() > 0) {
            $newFund = $row->accountbalance + $datax['amount'];
            $this->db->query('UPDATE user_accounts set accountbalance = :accountbalance WHERE  usertag = :usertag and  userid = :userid');

            $this->db->bind(':usertag', $datax['r_tag']);
            $this->db->bind(':userid', $datax['r_id']);
            $this->db->bind(':accountbalance', $newFund);
            if ($this->db->execute()) {
                return true;
            } else {
                $res = [
                    'status' => 401,
                    'message' => 'failed',
                ];
                print_r(json_encode($res));
                exit;
            }
        }else {
            return false;
        }


}
   public function  buyAirtime($datax)
  {


    $this->db->query('SELECT * FROM user_accounts WHERE  usertag = :usertag and  userid = :userid ');
    // Bind Valuesrid
    $this->db->bind(':usertag', $datax['tagname']);
    $this->db->bind(':userid', $datax['userid']);
    $row = $this->db->single();

    // Check roow
    if ($this->db->rowCount() > 0) {
      if ($row->accountbalance > $datax['amount']) {
         $newFund = $row->accountbalance - $datax['amount'];
           $this->db->query('UPDATE user_accounts set accountbalance = :accountbalance WHERE  usertag = :usertag and  userid = :userid');

      $this->db->bind(':usertag', $datax['tagname']);
      $this->db->bind(':userid', $datax['userid']);
      $this->db->bind(':accountbalance', $newFund);
      if ($this->db->execute()) {
        $this->db->query('INSERT INTO  vtu_services (userid, tagname,transaction_id, vtupackage, amount, date) VALUES( :userid, :tagname,:transaction_id, :vtupackage, :amount, :date)');
        // Bind Valuesrid
        $this->db->bind(':tagname', $datax['tagname']);
        $this->db->bind(':userid', $datax['userid']);
        $this->db->bind(':transaction_id', $datax['tr_id']);
        $this->db->bind(':vtupackage', $datax['vtupa']);
        $this->db->bind(':amount',"-".$datax['amount']);
        // Set the PHP timezone to GMT
      date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', date('Y-m-d H:i:s'));
        if ($this->db->execute()) {
            //return true;
             // Insert into all_transactions table
           $this->db->query('INSERT INTO  all_transactions (userid, tagname,transaction_id,transaction_type,transaction_status, vtupackage, amount, date, phone) VALUES( :userid, :tagname,:transaction_id,:transaction_type,:transaction_status, :vtupackage, :amount, :date, :phone)');
        // Bind Valuesrid
        $this->db->bind(':tagname', $datax['tagname']);
        $this->db->bind(':userid', $datax['userid']);
        $this->db->bind(':transaction_id', $datax['tr_id']);
        $this->db->bind(':transaction_type', $datax['tr_type']);
        $this->db->bind(':transaction_status', "successful");
        $this->db->bind(':vtupackage', $datax['vtupa']);
        $this->db->bind(':amount',$datax['amount']);
        $this->db->bind(':phone',$datax['phone']);
       // Set the PHP timezone to GMT
      date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', date('Y-m-d H:i:s'));
         if ($this->db->execute()) {
        //print_r($datax)
         // return true;
          return true;
      } else {
          return false;
      }
            
        } else {
               $res = [
              'status' => 401,
              'message' => 'failed 22',
            ];
            print_r(json_encode($res));
            exit;
        }
      } else {
          $res = [
              'status' => 401,
              'message' => 'failed',
            ];
            print_r(json_encode($res));
            exit;
      }
      }else {
          $res = [
              'status' => 401,
              'message' => 'not enough Balance',
            ];
            print_r(json_encode($res));
            exit;
      }
     
    
    }







  }


   public function  buyData($datax)
  {


    $this->db->query('SELECT * FROM user_accounts WHERE  usertag = :usertag and  userid = :userid ');
    // Bind Valuesrid
    $this->db->bind(':usertag', $datax['tagname']);
    $this->db->bind(':userid', $datax['userid']);
    $row = $this->db->single();

    // Check roow
    if ($this->db->rowCount() > 0) {
      if ($row->accountbalance > $datax['amount']) {
         $newFund = $row->accountbalance - $datax['amount'];
           $this->db->query('UPDATE user_accounts set accountbalance = :accountbalance WHERE  usertag = :usertag and  userid = :userid');

      $this->db->bind(':usertag', $datax['tagname']);
      $this->db->bind(':userid', $datax['userid']);
      $this->db->bind(':accountbalance', $newFund);
      if ($this->db->execute()) {
        $this->db->query('INSERT INTO  vtu_services (userid, tagname,transaction_id, vtupackage, amount, date) VALUES( :userid, :tagname,:transaction_id, :vtupackage, :amount, :date)');
        // Bind Valuesrid
        $this->db->bind(':tagname', $datax['tagname']);
        $this->db->bind(':userid', $datax['userid']);
        $this->db->bind(':transaction_id', $datax['tr_id']);
        $this->db->bind(':vtupackage', $datax['vtupa']);
        $this->db->bind(':amount', $datax['amount']);
        // Set the PHP timezone to GMT
      date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', date('Y-m-d H:i:s'));
        if ($this->db->execute()) {
                 // Insert into all_transactions table
       $this->db->query('INSERT INTO  all_transactions (userid, tagname,transaction_id, vtupackage, amount, transaction_type, date) VALUES( :userid, :tagname,:transaction_id, :vtupackage, :amount, :transaction_type, :date)');
      // Bind Valuesrid
       $this->db->bind(':fullname', $datax['fullname']);
      $this->db->bind(':veluxite_id', $datax['veluxite_id']);
      $this->db->bind(':bank_code', $datax['bank_code']);
      $this->db->bind(':bankname', $datax['bank_name']);
      $this->db->bind(':accountname', $datax['accname']);
      $this->db->bind(':amount', $datax['amount']);
      $this->db->bind(':tr_id', $datax['tr_id']);
      $this->db->bind(':transaction_type', 'vtu');
       // Set the PHP timezone to GMT
      date_default_timezone_set('Europe/Berlin'); // or 'Africa/Lagos'
     $this->db->bind(':date', date('Y-m-d H:i:s'));
         if ($this->db->execute()) {
        //print_r($datax)
         // return true;
          return true;
      } else {
          return false;
      }
           
           
        } else {
            return false;
        }
      } else {
          $res = [
              'status' => 401,
              'message' => 'failed',
            ];
            print_r(json_encode($res));
            exit;
      }
      }else {
          $res = [
              'status' => 401,
              'message' => 'not enough Balance',
            ];
            print_r(json_encode($res));
            exit;
      }
     
    
    }







  }
    public function getAccountBalance($id, $tag)
    {
        $this->db->query('SELECT * FROM user_accounts WHERE  usertag = :usertag and  userid = :userid ');
        // Bind Valuesrid
        $this->db->bind(':usertag', $tag);
        $this->db->bind(':userid', $id);
        $row = $this->db->single();
        if ($this->db->rowCount() > 0) {
            print_r($row->accountbalance);
        }
    }
    public function getaccountdetails()
    {
        $this->db->query('SELECT * FROM user_accounts ');
       
        $row = $this->db->resultSet();
        if ($this->db->rowCount() > 0) {
            print_r(json_encode($row));
        }
    }
  public function getUserTransactiony($id, $tag)
{
    // $this->db->query('SELECT * FROM transaction_history WHERE sender_tagname = :sender_tagname AND sender_id = :sender_id OR receiver_tagname = :receiver_tagname AND receiver_id = :receiver_id ORDER BY id desc');
    // $this->db->bind(':sender_tagname', $tag);
    // $this->db->bind(':sender_id', $id);
    // $this->db->bind(':receiver_tagname', $tag);
    // $this->db->bind(':receiver_id', $id);
    //  $row1 = $this->db->resultSet();
    //  $this->db->query('SELECT * FROM vtu_services WHERE userid = :userid');
         
    //  $this->db->bind(':userid', $id);
          
    //     $row2 = $this->db->resultSet(); 
    //     $this->db->query('SELECT * FROM withdrawal_record WHERE veluxite_id = :veluxite_id');
    //       $this->db->bind(':veluxite_id', $id);
    //       $row3 = $this->db->resultSet();
    //     $this->db->query('SELECT * FROM deposite_ WHERE veluxite_id = :veluxite_id');
    //       $this->db->bind(':veluxite_id', $id);
    //       $row4 = $this->db->resultSet();
    // if (isset($row1) || isset($row2) || isset($row3)) {
    //     print_r(json_encode($row1));
    //     print_r(json_encode($row2));
    //     print_r(json_encode($row3));
    //     print_r(json_encode($row4));
      
    // } else {
    //     print_r(('no transaction yet')) ;
    //     return false;
    // }
    $data = [
    "inapp" => $this->transactionhistory($id, $tag),
   "vtu" =>  $this->vtuservices($id, $tag),
   "withdrawal" => $this->withdrawalrecord($id, $tag),
    "deposit" => $this->deposite_($id, $tag)
    ];
    print_r(json_encode($data));
}


public function getUserTransactions($id, $tag)
{
    $this->db->query('
        SELECT * 
        FROM all_transactions 
        WHERE (sender_tagname = :sender_tagname AND sender_id = :sender_id) 
           OR (receiver_tagname = :receiver_tagname AND receiver_id = :receiver_id) OR userid = :userid OR veluxite_id = :veluxite_id 
        ORDER BY id DESC');

    $this->db->bind(':sender_tagname', $tag);
    $this->db->bind(':sender_id', $id);
    $this->db->bind(':receiver_tagname', $tag);
    $this->db->bind(':receiver_id', $id);
     $this->db->bind(':userid', $id);
   $this->db->bind(':veluxite_id', $id);
    $rows = $this->db->resultSet();
    

    if ($this->db->rowCount() > 0) {
        
//print_r($this->db->getBindings());
//print_r($rows);
         print_r(json_encode($rows));
    } else {
        echo 'Error: ' . $this->db->error();
        return 'no transaction made yet';
    }
}
public function getSubUserTransactions($id, $tag)
{
    $this->db->query('
        SELECT * 
        FROM sub_transaction_history 
        WHERE receiver_tagname = :receiver_tagname
        ORDER BY id DESC');
     
    $this->db->bind(':receiver_tagname', $tag);
    // $this->db->bind(':receiver_id', $id);
    $rows = $this->db->resultSet();
    


    if ($this->db->rowCount() > 0) {
        
//print_r($this->db->getBindings());
//print_r($rows);
         print_r(json_encode($rows));
    } else {
        
        print_r(json_encode("no transaction yet"));
    }
}


   public function transactionhistory($id, $tag)
{
   $this->db->query('SELECT * FROM transaction_history WHERE sender_tagname = :sender_tagname AND sender_id = :sender_id OR receiver_tagname = :receiver_tagname AND receiver_id = :receiver_id ORDER BY id desc');
    $this->db->bind(':sender_tagname', $tag);
    $this->db->bind(':sender_id', $id);
    $this->db->bind(':receiver_tagname', $tag);
    $this->db->bind(':receiver_id', $id);
    
     $rows = $this->db->resultSet();
    if ($this->db->rowCount() > 0) {
      return $rows ;
       
    } else {
        return (('no inapp transaction yet')) ;
        // return false;
    }
}
   public function sentUser($id, $tag)
{
   $this->db->query('SELECT * FROM transaction_history WHERE sender_tagname = :sender_tagname AND sender_id = :sender_id ORDER BY id desc');
    $this->db->bind(':sender_tagname', $tag);
    $this->db->bind(':sender_id', $id);
     $rows = $this->db->resultSet();
    if ($this->db->rowCount() > 0) {
      return $rows ;
       
    } else {
        return (('no transaction yet')) ;
        // return false;
    }
}
   public function vtuservices($id, $tag)
{
  $this->db->query('SELECT * FROM vtu_services WHERE userid = :userid');
         
     $this->db->bind(':userid', $id);
          
        $rows = $this->db->resultSet(); 
    if ($this->db->rowCount() > 0) {
        // print_r(json_encode($rows));
        return $rows ;
       
    } else {
        return (('no vtuservice transaction yet')) ;
        // return false;
    }
}
   public function withdrawalrecord($id, $tag)
{
    $this->db->query('SELECT * FROM withdrawal_record WHERE veluxite_id = :veluxite_id');
          $this->db->bind(':veluxite_id', $id);
          $rows = $this->db->resultSet();
    if ($this->db->rowCount() > 0) {
      return $rows ;
       
    } else {
        return (('no withdrawal transaction yet')) ;
        // return false;
    }
}
   public function deposite_($id, $tag)
{
    $this->db->query('SELECT * FROM deposite_ WHERE veluxite_id = :veluxite_id');
          $this->db->bind(':veluxite_id', $id);
          $rows = $this->db->resultSet();
    if ($this->db->rowCount() > 0) {
        return $rows ;
       
    } else {
        return (('no deposit transaction yet')) ;
        // return false;
    }
}
    public function getTagname()
    {
        $this->db->query('SELECT * FROM initkeyrid');
        $rows = $this->db->resultSet();
        if ($this->db->rowCount() > 0) {
            print_r(json_encode($rows));
        } else {
            return false;
        }
    }
    public function getallTransactions()
    {
        $this->db->query('SELECT * FROM transaction_history');
        $rows = $this->db->resultSet();
        if ($this->db->rowCount() > 0) {
            print_r(json_encode($rows));
        } else {
            return false;
        }
    }
    public function getSubuser($id)
    {
        $this->db->query('SELECT * FROM sub_initkeyrid  WHERE registrer_id =:registrer_id ');
        $this->db->bind(':registrer_id', $id);
        $rows = $this->db->resultSet();
        if ($this->db->rowCount() > 0) {
            print_r(json_encode($rows));
        } else {
            return false;
        }
    }
   
    public function deleteSubuser($id, $tag)
    {
        $this->db->query('DELETE  FROM sub_initkeyrid  WHERE veluxite_id =:veluxite_id AND user_tag =:user_tag');
        $this->db->bind(':veluxite_id', $id);
        $this->db->bind(":user_tag" ,$tag);
        $rows = $this->db->resultSet();
        if ($this->db->execute()) {
            $res = [
                'status' => "true",
                "message" => "deleted sucessfully"
                ];
            print_r(json_encode($res));
        } else {
            return false;
        }
    }
    public function getName($name)
    {
        $this->db->query('SELECT full_name FROM initkeyrid WHERE user_tag =:user_tag ');
         $this->db->bind(':user_tag', $name);
        $rows = $this->db->resultSet();
        if ($this->db->rowCount() > 0) {
            print_r(json_encode($rows));
        } else {
            return false;
        }
    }
    
    /////Added by Divine 23-06-21
       public function getImage($name)
    {
        $this->db->query('SELECT image FROM initkeyrid WHERE user_tag =:user_tag ');
         $this->db->bind(':user_tag', $name);
      // $rows = $this->db->resultSet();
       $rows = $this->db->single();
       
        // if ($this->db->rowCount() > 0) {
        //     print_r(json_encode($rows));
        // } else {
        //     return false;
        // }
           if ($rows) {
            print_r(json_encode($rows));
        } else {
            return false;
        }
    }
public function getallusers()
    {
        $this->db->query('SELECT * FROM initkeyrid');
        $rows = $this->db->resultSet();
        if ($this->db->rowCount() > 0) {
            print_r(json_encode($rows));
        } else {
            return false;
        }
    }
 
}

